# Stored Requests

This document gives a technical overview of the Stored Requests feature.

Docs outlining the motivation and uses will be added sometime in the future.

## Quickstart

Configure your server to read stored requests from the filesystem:

```yaml
stored_requests:
  filesystem: true
```

Choose an ID to reference your stored request data. Throughout this doc, replace {id} with the ID you've chosen.

Add the file `stored_requests/data/by_id/{id}.json` and populate it with some [Imp](https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf#page=17) data.

```json
{
  "id": "test-imp-id",
  "banner": {
    "format": [
      {
        "w": 300,
        "h": 250
      },
      {
        "w": 300,
        "h": 600
      }
    ]
  },
  "ext": {
    "appnexus": {
      "placementId": 10433394
    }
  }
}
```

Start your server.

```bash
go build .
./prebid-server
```

And then `POST` to [`/openrtb2/auction`](../endpoints/openrtb2/auction.md) with your chosen ID.

```json
{
  "id": "test-request-id",
  "imp": [
    {
      "ext": {
        "prebid": {
          "storedrequest": {
            "id": "{id}"
          }
        }
      }
    }
  ]
}
```

The auction will occur as if the HTTP request had included the content from `stored_requests/data/by_id/{id}.json` instead.

## Partially Stored Requests

You can also store _part_ of the Imp on the server. For example:

```json
{
  "banner": {
    "format": [
      {
        "w": 300,
        "h": 250
      },
      {
        "w": 300,
        "h": 600
      }
    ]
  },
  "ext": {
    "appnexus": {
      "placementId": 10433394
    }
  }
}
```

This is not _fully_ legal OpenRTB `imp` data, since it lacks an `id`.

However, incoming HTTP requests can fill in the missing data to complete the OpenRTB request:

```json
{
  "id": "test-request-id",
  "imp": [
    {
      "id": "test-imp-id",
      "ext": {
        "prebid": {
          "storedrequest": {
            "id": "{id}"
          }
        }
      }
    }
  ]
}
```

If the Stored Request and the HTTP request have conflicting properties,
they will be resolved with a [JSON Merge Patch](https://tools.ietf.org/html/rfc7386).
HTTP request properties will overwrite the Stored Request ones.

## Stored BidRequests

So far, our examples have only used Stored Imp data. However, Stored Requests
are also allowed on the [BidRequest](https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf#page=15).
These work exactly the same way, but support storing properties like timeouts and price granularity.

For example, assume the following `stored_requests/data/by_id/stored-request.json`:

```json
{
    "tmax": 1000,
    "ext": {
      "prebid": {
        "targeting": {
          "pricegraularity": "low",
        }
      }
    }
  }
```

Then an HTTP request like:

```json
{
  "id": "test-request-id",
  "imp": [
    "Any valid Imp data in here"
  ],
  "ext": {
    "prebid": {
      "storedrequest": {
        "id": "stored-request.json"
      }
    }
  }
}
```

will produce the same auction as if the HTTP request had been:

```json
{
  "id": "test-request-id",
  "tmax": 1000,
  "imp": [
    "Any valid Imp data in here"
  ],
  "ext": {
    "prebid": {
      "targeting": {
        "pricegraularity": "low",
      }
    }
  }
}
```

Prebid Server does allow Stored BidRequests and Stored Imps in the same HTTP Request.
The Stored BidRequest will be applied first, and then the Stored Imps after.

**Beware**: Stored Request data will not be applied recursively.
If a Stored BidRequest includes Imps with their own Stored Request IDs,
then the data for those Stored Imps not be resolved.


## Alternate backends

Stored Requests do not need to be saved to files. [Other backends](../../stored_requests/backends) are supported
with different [configuration options](configuration.md). For example:

```yaml
stored_requests:
  postgres:
    host: localhost
    port: 5432
    user: db-username
    dbname: database-name
    query: SELECT id, requestData FROM stored_requests WHERE id IN %ID_LIST%;
```

If you need support for a backend that you don't see, please [contribute it](contributing.md).
