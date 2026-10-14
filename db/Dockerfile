FROM postgres:18
# Build contexts are uploaded to remote Docker; host bind mounts are not.
COPY init/ /docker-entrypoint-initdb.d/
RUN chmod +x /docker-entrypoint-initdb.d/*.sh
