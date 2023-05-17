# Stored Requests

See https://docs.prebid.org/prebid-server/features/pbs-storedreqs.html

This document gives a technical overview of the Stored Requests feature in PBS-Go.

Docs outlining the motivation and uses will be added sometime in the future.

## Quickstart

Configure your server to read stored requests from the filesystem:

```yaml
stored_requests:
  filesystem: true
```

Choose an ID to reference your stored request data. Throughout this doc, replace {id} with the ID you've chosen.

Add the file `stored_requests/data/by_id/stored_imps/{id}.json` and populate it with some [Imp](https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf#page=17) data.

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
      "placementId": 12883451
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

The auction will occur as if the HTTP request had included the content from `stored_requests/data/by_id/stored_imps/{id}.json` instead.

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
      "placementId": 12883451
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

For example, assume the following `stored_requests/data/by_id/stored_requests/stored-request.json`:

```json
{
    "tmax": 1000,
    "ext": {
      "prebid": {
        "targeting": {
          "pricegranularity": "low",
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
        "id": "stored-request"
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
        "pricegranularity": "low",
      }
    }
  }
}
```

Prebid Server does allow Stored BidRequests and Stored Imps in the same HTTP Request.
The Stored BidRequest patch will be applied first, and then the Stored Imp patches after.

**Beware**: Stored Request data will not be applied recursively.
If a Stored BidRequest includes Imps with their own Stored Request IDs,
then the data for those Stored Imps not be resolved.

## Alternate backends

Stored Requests do not need to be saved to files. [Other backends](../../stored_requests/backends) are supported
with different [configuration options](configuration.md). For example:

```yaml
stored_requests:
  database:
    connection:
      driver: postgres
      host: localhost
      port: 5432
      user: db-username
      dbname: database-name
    fetcher:
      query: SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in $REQUEST_ID_LIST UNION ALL SELECT id, impData, 'imp' as type FROM stored_imps WHERE id in $IMP_ID_LIST;
```

### Supported Databases
- postgres
- mysql

### Query Syntax
All database queries should be expressed using the native SQL syntax of your supported database of choice with one caveat.

For all supported database drivers, wherever you need to specify a query parameter, you must not use the native syntax (e.g. `$1`, `%%`, `?`, etc.), but rather a PBS-specific syntax to represent the parameter which is of the format `$VARIABLE_NAME`. PBS currently supports just four query parameters, each of which pertains to particular config queries, and here is how they should be specified in your queries:
- last updated at timestamp --> `$LAST_UPDATED`
- stored request ID list --> `$REQUEST_ID_LIST`
- stored imp ID list --> `$IMP_ID_LIST`
- stored response ID list --> `$ID_LIST`

See the query defined at `stored_requests.database.connection.fetcher.query` in the yaml config above as an example of how to mix these variables in with native SQL syntax.

```yaml
stored_requests:
  http:
    endpoint: http://stored-requests.prebid.com
    amp_endpoint: http://stored-requests.prebid.com?amp=true

```

If you need support for a backend that you don't see, please [contribute it](contributing.md).

## Caches and Event-based updating

Stored Request data can also be cached or updated while PBS is running.
Conceptually, Stored Request data is managed by three separate interfaces in the code:

**Fetcher**: These pull data directly from a backend.
**Cache**: Duplicates data which the Fetcher _could_ find so that it can be accessed more quickly.
**EventProducer**: Returns some Channels which can be used to signal changes to Stored Request data.

Fetchers, Caches, and EventProducers can also be chosen in the the app config.
At least one Fetcher is _required_ to make use of Stored Requests.

If more than one Fetcher is defined, they will be ordered and used as fallback data sources.
This isn't a great idea for Prod in the long-term, but may be useful temporarily if you're trying
to transition from one backend to another.

If more than one Cache is defined, they will be composed into a single Cache. Saves will propagate to all Cache layers.
Any concrete Fetcher in the project will be composed with any Cache(s) to create a new Fetcher.

EventProducer events are used to Save or Invalidate values from the Cache(s).
Saves and invalidates will propagate to all Cache layers.

Here is an example `pbs.yaml` file which looks for Stored Requests first from Database (i.e. Postgres), and then from an HTTP endpoint.
It will use an in-memory LRU cache to store data locally, and poll another HTTP endpoint to listen for updates.

```yaml
stored_requests:
  database:
    connection:
      driver: postgres
      host: localhost
      port: 5432
      user: db-username
      dbname: database-name
    fetcher:
      query: SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in $REQUEST_ID_LIST UNION ALL SELECT id, impData, 'imp' as type FROM stored_imps WHERE id in $IMP_ID_LIST;
  http:
    endpoint: http://stored-requests.prebid.com
    amp_endpoint: http://stored-requests.prebid.com?amp=true
  in_memory_cache:
    ttl_seconds: 300 # 5 minutes
    request_cache_size_bytes: 107374182 # 0.1GB
    imp_cache_size_bytes: 107374182 # 0.1GB
  http_events:
    endpoint: http://stored-requests.prebid.com
    amp_endpoint: http://stored-requests.prebid.com?amp=true
    refresh_rate_seconds: 60
    timeout_ms: 100
```

Pull Requests for new Fetchers, Caches, or EventProducers are always welcome.
