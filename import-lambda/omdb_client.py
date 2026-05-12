import requests


class OMDBClient:
    def __init__(self, base_url: str, api_key: str, timeout: int = 10):
        self.base_url = base_url
        self.api_key = api_key
        self.timeout = timeout

    def search(self, term: str, page: int = 1) -> list:
        params = {"apikey": self.api_key, "s": term, "page": page}
        resp = requests.get(self.base_url, params=params, timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()
        if data.get("Response") != "True":
            raise Exception(f"OMDB search failed for term '{term}': {data.get('Error', 'unknown')}")
        return data.get("Search", [])
