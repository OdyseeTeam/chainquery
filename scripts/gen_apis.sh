#!/usr/bin/env bash
(
    #download swagger-codegen-cli if not already there.
    cd "$PWD"
    mkdir -p $PWD/swagger
    if [ ! -f $PWD/swagger/swagger-codegen-cli.jar ]; then
        echo "swagger cli not found, downloading..."
        wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.8/swagger-codegen-cli-2.4.8.jar -O ./swagger/swagger-codegen-cli.jar
    fi
    #Generate API docs

    executable="$PWD/swagger/swagger-codegen-cli.jar"

    export JAVA_OPTS="${JAVA_OPTS} -XX:MaxPermSize=256M -Xmx1024M -DloggerPath=conf/log4j.properties"
    agsDocs="$@ generate -i $PWD/swagger/chainquery.yaml -l dynamic-html  -o $PWD"
    agsServer="$@ generate -i $PWD/swagger/chainquery.yaml -l go-server -t $PWD/swagger/modules/go-server -Dmodel={}  -o $PWD/swagger/apiserver"
    agsClient_go="$@ generate -i $PWD/swagger/chainquery.yaml -l go  -o $PWD/swagger/clients/goclient"
    agsClient_python="$@ generate -i $PWD/swagger/chainquery.yaml -l python  -o $PWD/swagger/clients/pythonclient"


    java $JAVA_OPTS -jar $executable $agsDocs

    java $JAVA_OPTS -jar $executable $agsServer

    java $JAVA_OPTS -jar $executable $agsClient_go
    java $JAVA_OPTS -jar $executable $agsClient_python

)