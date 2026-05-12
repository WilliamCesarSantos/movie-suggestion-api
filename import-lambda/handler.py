import json
import os
from omdb_client import OMDBClient
from sqs_publisher import SQSPublisher

omdb_client = OMDBClient(
    base_url=os.environ.get("OMDB_BASE_URL", "http://www.omdbapi.com"),
    api_key=os.environ.get("OMDB_API_KEY", ""),
)

sqs_publisher = SQSPublisher(
    queue_url=os.environ.get("SQS_QUEUE_URL", ""),
    region=os.environ.get("AWS_REGION", "us-east-1"),
    endpoint_url=os.environ.get("AWS_ENDPOINT", None),
)

def lambda_handler(event, context):
    search_terms = event.get("searchTerms", [])
    max_pages = event.get("maxPages", 1)
    
    total_published = 0
    errors = []
    
    for term in search_terms:
        for page in range(1, max_pages + 1):
            try:
                results = omdb_client.search(term, page)
                for result in results:
                    sqs_publisher.publish(result["imdbID"])
                    total_published += 1
            except Exception as e:
                errors.append({"term": term, "page": page, "error": str(e)})
    
    return {
        "published": total_published,
        "errors": errors,
    }
