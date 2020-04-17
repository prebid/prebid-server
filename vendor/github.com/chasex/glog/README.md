# glog

Package glog implements a simple level logging package based on golang's
standard [log](github.com/golang/go/tree/master/src/log) 
and [glog](github.com/golang/glog) package. 
It has fully compatible interface to standard log package. 
It defines a type, Logger, with methods for formatting output. 

Basic examples:

    options := glog.LogOptions{
    	File: "./abc.log",
    	Flag: glog.LstdFlags,
    	Level: glog.Ldebug,
    	Mode: glog.R_None,
    }
    logger, err := glog.New(options)
    if err != nil {
    	panic(err)
    }
    logger.Debug("hello world")
    logger.Infof("hello, %s", "chasex")
    logger.Warn("testing message")
    logger.Flush()

The output contents in abc.log will be:

    2016/02/16 17:50:07 DEBUG hello world
    2016/02/16 17:50:07 INFO hello, chasex
    2016/02/16 17:50:07 WARN testing message

It also support rotating log file by size, hour or day.
According to rotate mode, log file name has distinct suffix:

    R_None: no suffix, just base name, abc.log.
    R_Size: suffix with date and clock, abc.log-YYYYMMDD-HHMMSS.
    R_Hour: suffix with date and hour, abc.log-YYYYMMDD-HH.
    R_Day:  suffix with date, abc.log-YYYYMMDD.

Note that it has a daemon routine flushing buffered data to underlying file
periodically (default every 30s). When exit, remember calling Flush() manually,
otherwise it may cause some date loss.

For more details see document: https://godoc.org/github.com/chasex/glog
