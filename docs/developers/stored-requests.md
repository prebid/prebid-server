# Stored Requests

This document gives a technical overview of the Stored Impressions feature.

Docs outlining the motivation and uses will be added sometime in the future.

## Quickstart

Configure your server to read stored requests from the filesystem:

```yaml
ortb2_config:
  filesystem: true
```

Choose an ID to reference your request. Throughout this doc, replace {id} with the ID you've chosen.

Add the file `openrtb2_configs/for_requests/{id}.json` and populate it with some Impression data.

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
          "managedconfig": {
            "id": "{id}"
          }
        }
      }
    }
  ]
}
```

The Auction will occur as if the Request had used the content from `openrtb2_configs/for_requests/{id}.json` instead.

## Partially Stored Requests

You can also store _part_ of the Impression on the server. For example:

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

Note that OpenRTB requires each `imp` to have an `id` property.

For a given HTTP Request to be valid, it must contain these missing properties:

```json
{
    "id": "test-request-id",
    "imp": [
      {
        "id": "test-imp-id",
        "ext": {
          "prebid": {
            "managedconfig": {
              "id": "{id}"
            }
          }
        }
      }
    ]
  }
```

If the Stored Request and the HTTP Request have conflicting properties,
they will be resolved with a [JSON Merge Patch](https://tools.ietf.org/html/rfc7386).
HTTP Request properties will overwrite the Stored Request ones.

## Alternate backends

Stored Requests do not need to be saved to files. [Other backends](../../openrtb2_config/) can be selected
with different [Configuration options](configuration.yaml).

```yaml
ortb2_config:
  postgres:
    host: localhost
    port: 5432
    user: db-username
    dbname: database-name
    query: SELECT id, config FROM some_table WHERE id IN %ID_LIST%;
```

If you need a backend that you don't see, please [contribute it](contributing.md).
