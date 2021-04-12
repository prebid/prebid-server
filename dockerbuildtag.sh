#!/bin/bash

# kako buildati prebid-server-java
# !pazi, ko je intellijidea up, je premalo memorya. zato jo prej IZKLOPIŠ!



GITLABPWD=$(</home/uporabnik/.gesla/gitlabpwd.txt)
GITLAB_ID_USER=$(</home/uporabnik/.gesla/gitlabusr.txt)
DESTINATIONREGISTRY=registry.gitlab.com
GITLABSUBFOLDER=dockerimagesl/
#GITLABTAG=20190123
GITLABTAG=$(date +%F_%H%M%S)
PROGRAMTYPE="linux"
IMAGENAME="prebid-server-go"
GITLABSUBFOLDER="dockerimagesl/"




# mvn clean package -e
#mvn clean package -Dmaven.test.skip=true -Dcheckstyle.skip=true

#s tem dobiš novi jar:
#Building jar: /home/uporabnik/prebid-java/prebid-server-java/target/prebid-server.jar

#potem lahko zbuildaš z dockerjem
docker build .
docker build -t myappmediaserver/prebid-server-go:latest .




                echo "using gitlab subfolder: ${GITLABSUBFOLDER} for ${PROGRAMTYPE} imagename:${IMAGENAME}"

                docker login ${DESTINATIONREGISTRY} -u $GITLAB_ID_USER -p $GITLABPWD
                echo "credentials used for ${DESTINATIONREGISTRY} -u $GITLAB_ID_USER -p $GITLABPWD "
                docker tag myappmediaserver/${IMAGENAME} ${DESTINATIONREGISTRY}/${GITLAB_ID_USER}/${GITLABSUBFOLDER}${IMAGENAME}:${GITLABTAG}
                docker tag myappmediaserver/${IMAGENAME} ${DESTINATIONREGISTRY}/${GITLAB_ID_USER}/${GITLABSUBFOLDER}${IMAGENAME}:latest
                echo "publishing"
                docker push ${DESTINATIONREGISTRY}/${GITLAB_ID_USER}/${GITLABSUBFOLDER}${IMAGENAME}:${GITLABTAG}
                docker push ${DESTINATIONREGISTRY}/${GITLAB_ID_USER}/${GITLABSUBFOLDER}${IMAGENAME}:latest
                echo "publish completed. now logging out"
                docker logout ${DESTINATIONREGISTRY}
                echo "published..."
                LIVEPUBLISHED=true


NAMESPACE=ssp
kubectl rollout restart deployment prebidservergo -n ${NAMESPACE}

sleep 5