import urllib3
import urllib3
import json
import hashlib
from jsonpath import JSONPath
import base64
import os
from twilio.rest import Client

handler = urllib3.PoolManager()
SECRET_KEY = os.environ["KEY"]


def latest_data(suburb):
    response = handler.request(
        "GET",
        "https://discover.data.vic.gov.au/api/3/action/datastore_search?resource_id=afb52611-6061-4a2b-9110-74c920bede77",
    )
    data = search_data(json.loads(response.data), suburb)
    return data, hashlib.md5(json.dumps(data).encode("utf-8")).hexdigest()


def search_data(data, postcode):
    return JSONPath(f'$.result.records[?(@.Suburb=="{postcode}")]').parse(data)


def get_hash(key="hash"):
    response = handler.request(
        "GET",
        f"https://keyvalue.immanuel.co/api/KeyVal/GetValue/{SECRET_KEY}/{key}",
    )
    return json.loads(response.data)


def set_hash(value, key="hash"):
    response = handler.request(
        "POST",
        f"https://keyvalue.immanuel.co/api/KeyVal/UpdateValue/{SECRET_KEY}/{key}/{value}",
    )
    return json.loads(response.data)


def notify():
    account_sid = get_hash("sid")
    auth_token = get_hash("token")

    client = Client(account_sid, auth_token)

    client.api.account.messages.create(
        to="+61432071731",
        from_="exposure",
        body="There are new Docklands exposure sites",
    )


if __name__ == "__main__":
    data, hash = latest_data("Docklands")
    if hash != get_hash():
        set_hash(hash)
        notify()