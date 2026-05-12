import boto3
import json


class SQSPublisher:
    def __init__(self, queue_url: str, region: str = "us-east-1", endpoint_url: str = None):
        kwargs = {"region_name": region}
        if endpoint_url:
            kwargs["endpoint_url"] = endpoint_url
        self.client = boto3.client("sqs", **kwargs)
        self.queue_url = queue_url

    def publish(self, imdb_id: str):
        message = json.dumps({"imdbId": imdb_id})
        self.client.send_message(
            QueueUrl=self.queue_url,
            MessageBody=message,
        )
