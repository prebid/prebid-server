#!/bin/bash

# This script is mostly copy-pasta from unicorn-shepherd.sh
#
# This script is a bridge between upstart and graceful go processes
#
# The reason this is necessary is that upstart wants to start and watch a pid for its entire
# lifecycle. However, grace's cool no-downtime restart feature creates a new grace golang process
# which will take over the socket/port from, and then kill, the original process. This makes
# upstart think that the service died and (especially with `respawn`) it gets wonky from there.
#
# So this script is started by upstart. It can detect if a go process is already running
# and will wait for it to exit.  Then upstart will restart this script which will see if
# a process is running again. On no-downtime restarts it will find the new process
# and wait on it to take over, and so on.  So grace is managing its own lifecycle and this script
# gives upstart a single pid to start and watch.
#
# This script also handles the signals sent by upstart to stop and restart and sends them to the
# running go process to initiate a no-downtime restart when the upstart 'restart' command
# is given to this service.
#
# We do some crazy magic in is_restarting to determine if we are restarting or stopping.


#############################################################
##
## Set up environment
##
#############################################################

COMMAND=$1
SERVICE=$2

# logs to syslog with service name and the pid of this script
log() {
    # we have to send this to syslog ourselves instead of relying on whoever launched
    # us because the exit signal handler log output never shows up in the output stream
    # unless we do this explicitly
    echo "$@" | logger -t "${SERVICE}[$$]"
}

# assume upstart config cd's us into project root dir.
BASE_DIR=$PWD
TRY_RESTART=true

#############################################################
##
## Support functions
##
#############################################################

# Bail out if all is not well
check_environment(){
    if [ "x" = "x${COMMAND}" ] ; then
        log "Missing required argument: Command to launch."
        exit 1
    fi

    if [ "x" = "x${SERVICE}" ] ; then
        log "Missing required second argument: Upstart service name that launched this script"
        exit 1
    fi

    # default to APP_ENV if RACK_ENV isn't set
    export RACK_ENV="${RACK_ENV:-$APP_ENV}"

    if [ ! -n "$RACK_ENV" ] ; then
        log "Neither RACK_ENV nor APP_ENV environment variable are set. Exiting."
        exit 1
    fi

}

# Return the pid of the new go process. If there are go processes running, not
# a new one and one marked old which is exiting, but two that think they are current
# then exit with an error. How could we handle this better? When would it happen?
# Delete any pid files found which have no corresponding running processes.
process_pid() {
    local pid=''
    local extra_pids=''
    local multi_master=false

    for PID_FILE in $(find $BASE_DIR/pids/ -name "*.pid") ; do
        local p=`cat ${PID_FILE}`

        if is_pid_running $p ; then
            if [ -n "$pid" ] ; then
                multi_master=true
                extra_pids="$extra_pids $p"
            else
                pid="$p"
            fi
        else
            log "Deleting ${COMMAND} pid file with no running process '$PID_FILE'"
            rm $PID_FILE 2> /dev/null  || log "Failed to delete pid file '$PID_FILE': $!"
        fi
    done
    if $multi_master ; then
        log "Found more than one not old ${COMMAND} process running. Pids are '$pid $extra_pids'."
        log "Killing them all and restarting."
        kill -9 $pid $extra_pids
        exit 1
    fi

    echo $pid
    # return status so we can use this function to see if the process is running
    [ -n "$pid" ]
}

is_pid_running() {
    local pid=$1
    if [ ! -n "$pid" ] || ! [ -d "/proc/$pid" ] ; then
        return 1
    fi
    return 0
}


# output parent process id of argument
ppid() {
    ps -p $1 -o ppid=
}

free_mem() {
    free -m | grep "buffers/cache:" | awk '{print $4};'
}

# This is the on exit handler. It checks if we are restarting or not and either sends USR1+USR2
# signal to grace or, if the service is being stopped, kill the grace process.
respawn_new_process() {
    # TRY_RESTART is set to false on exit where we didn't recieve TERM.
    # When we used "trap command TERM" it did not always trap propertly
    # but "trap command EXIT" runs command every time no matter why the script
    # ends. So we set this env var to false if we don't need to respawn which is if grace
    # dies by itself or is restarted externally, usually through the deploy script
    # or we never succesfully started it.
    # If we receive a TERM, like from upstart on stop/restart, this won't be set
    # and we'll send USR1+USR2 to restart the go server.
    if $TRY_RESTART ; then
        if is_service_in_state "restart" ; then
            local pid=`process_pid`
            if [ -n "$pid" ] ; then
                # free memory before restart. Restart is unreliable with not enough memory.
                # New process crashes during startup etc.
                let min_mem=1000

                if [ `free_mem` -lt $min_mem ] ; then
                    log "Not enough memory to restart. Killing the process and allowing upstart to restart."
                    kill -9 ${pid}
                else
                  # by sending USR1 you are instructing the process to *expect* the graceful restart process
                  # and not trap errors when killed.
                  kill -USR1 ${pid}
                  # gracefully stop serving from the old process, and allow a new one to come up
                  kill -USR2 ${pid}
                  log "Respawn signals HUP + USR1 + USR2 sent to ${COMMAND} process ${pid}"
                fi
            else
                log "No ${COMMAND} process found. Exiting. A new one will launch when we are restarted."
            fi
        elif is_service_in_state "stop" ; then
            local pid=`process_pid`
            if [ -n "$pid" ] ; then
                tries=1
                while is_pid_running ${pid} && [ $tries -le 5 ] ; do
                    log "Service is STOPPING. Trying to kill '${COMMAND}' at pid '${pid}'. Try ${tries}"
                    # send USR1 first, so we don't panic
                    kill -USR1 ${pid}
                    kill ${pid}
                    tries=$(( $tries + 1 ))
                    sleep 1
                done

                if is_pid_running ${pid} ; then
                    log "Done waiting for '${COMMAND}' process '${pid}' to die. Killing for realz"
                    # send USR1 first, so we don't panic
                    kill -USR1 ${pid}
                    kill -9 ${pid}
                else
                    log "${COMMAND} process '${pid}' is dead."
                fi
            fi
        else
            log "Service is neither stopping nor restarting. Exiting."
        fi
    else
        log "Not checking for restart"
    fi
}

# Upstart does not have the concept of "restart". When you restart a service it is simply
# stopped and started. But this defeats the purpose of grace's USR2 no downtime trick.
# So we check the service states of the foreman exported services. If any of them are
# start/stopping or start/post-stop it means that they are stopping but that the service
# itself is still schedule to run. This means restart. We can use this to differentiate between
# restarting and stopping so we can signal grace to restart or actually kill it appropriately.
is_service_in_state() {
    local STATE=$1
    if [ "$STATE" = "restart" ] ; then
        PATTERN="(start/stopping|start/post-stop)"
    elif [ "$STATE" = "stop" ] ; then
        PATTERN="/stop"
    else
        log "is_service_in_state: State must be one of 'stop' or 'restart'. Got '${STATE}'"
        exit 1
    fi
    # the service that started us and the foreman parent services, pruning off everything
    # after each successive dash to find parent service
    # e.g. myservice-web-1 myservice-web myservice
    services=( ${SERVICE} ${SERVICE%-*} ${SERVICE%%-*} )

    IN_STATE=false

    for service in "${services[@]}" ; do
        if /sbin/status ${service} | egrep -q "${PATTERN}" ; then
            log "Service ${service} is in state '${STATE}'. - '$(/sbin/status ${service})'"
            IN_STATE=true
        fi
    done

    $IN_STATE # this is the return code for this function
}

#############################################################
##
## Trap incoming signals
##
#############################################################

# trap TERM which is what upstart uses to both stop and restart (stop/start)
trap "respawn_new_process" EXIT

#############################################################
##
## Main execution
##
#############################################################

check_environment


if ! process_pid ; then

    log "No ${COMMAND} process found. Launching new ${COMMAND} process in env '$RACK_ENV' in directory '$BASE_DIR'"

    # setsid to start this process in a new session because when upstart stops or restarts
    # a service it kills the entire process group of the service and relaunches it. Because
    # we are managing grace separately from upstart it needs to be in its own
    # session (group of process groups) so that it survives the original process group
    setsid ${COMMAND} 2>&1 &

    tries=1
    while [ $tries -le 10 ] && ! process_pid ; do
        log "Waiting for the new go process to launch"
        tries=$(( $tries + 1 ))
        sleep 1
    done
fi

PID=`process_pid`

if is_pid_running $PID ; then
    # hang out while the grace process is alive. Once its gone we will exit
    # this script. When upstart respawns us we will end up in the if statement above
    # to relaunch a new grace process.
    log "Found running ${COMMAND} master $PID. Awaiting its demise..."
    while is_pid_running $PID ; do
        sleep 5
    done
    log "${COMMAND} master $PID has exited."
else
    log "Failed to start ${COMMAND} master. Will try again on respawn. Exiting"
fi

TRY_RESTART=false
