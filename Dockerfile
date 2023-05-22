FROM sonatype/nexus-platform-cli:0.0.20190220-163049.9ebe2a7

ENV SONATYPE_DIR=/opt/sonatype 

COPY NexusPublisher.groovy ${SONATYPE_DIR}/bin/

CMD ["sh", "-c", "groovy ${SONATYPE_DIR}/bin/NexusPublisher.groovy --username ${PLUGIN_USERNAME} --password ${PLUGIN_PASSWORD} \
    --serverurl=${PLUGIN_SERVER_URL} --filename=${PLUGIN_FILENAME} --format=${PLUGIN_FORMAT} --repository=${PLUGIN_REPOSITORY} ${PLUGIN_ATTRIBUTES}"]