#!/usr/bin/env bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

(
    #download swagger-codegen-cli if not already there.
    cd "$DIR"
    mkdir -p $DIR/swagger
    if [ ! -f $DIR/swagger/swagger-codegen-cli.jar ]; then
        echo "swagger cli not found, downloading..."
        wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.2.3/swagger-codegen-cli-2.2.3.jar -O ./swagger/swagger-codegen-cli.jar
    fi
    #Generate API docs

    executable="./swagger/swagger-codegen-cli.jar"

    export JAVA_OPTS="${JAVA_OPTS} -XX:MaxPermSize=256M -Xmx1024M -DloggerPath=conf/log4j.properties"
    agsDocs="$@ generate -i $DIR/chainquery.yaml -l dynamic-html  -o $DIR"
    agsServer="$@ generate -i $DIR/chainquery.yaml -l go-server -t $DIR/swagger/modules/go-server -DpackageName=chainqueryapis -o swagger/apiserver"
    agsClient_go="$@ generate -i $DIR/chainquery.yaml -l go  -o swagger/clients/go"
    agsClient_python="$@ generate -i $DIR/chainquery.yaml -l python  -o swagger/clients/python"


    java $JAVA_OPTS -jar $executable $agsDocs

    java $JAVA_OPTS -jar $executable $agsServer

    java $JAVA_OPTS -jar $executable $agsClient_go
    java $JAVA_OPTS -jar $executable $agsClient_python

)