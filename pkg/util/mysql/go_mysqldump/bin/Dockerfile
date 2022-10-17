FROM mysql:8
RUN apt-get update && apt-get install -y time

COPY go_mysqldump_linux /bin/go_mysqldump

ENTRYPOINT ["sh"]
CMD ["-c", "echo do magic stuff"]

