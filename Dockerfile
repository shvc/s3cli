FROM alpine:3.15.6

COPY s3cli /usr/bin/

ENV S3_REGION=cn2 \
	S3_ENDPOINT=http://127.0.0.1:9000 \
	AWS_ACCESS_KEY=root \
	AWS_SECRET_KEY=ChangeMe

ENTRYPOINT ["/usr/bin/s3cli"]