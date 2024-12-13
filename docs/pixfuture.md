# Pixfuture Adapter

## Features

| Feature                   | Value               | Feature                   | Value               |
|---------------------------|---------------------|---------------------------|---------------------|
| **Bidder Code**           | `pixfuture`         | **Prebid.org Member**     | No                 |
| **Prebid.js Adapter**     | Yes                 | **Prebid Server Adapter** | Yes                |
| **Media Types**           | Display             | **Multi Format Support**  | Will-not-bid       |
| **TCF-EU Support**        | Yes                 | **IAB GVL ID**            | TBD                |
| **GPP Support**           | USState_All         | **DSA Support**           | Check with bidder  |
| **USP/CCPA Support**      | Yes                 | **COPPA Support**         | Yes                |
| **Supply Chain Support**  | Yes                 | **Demand Chain Support**  | Check with bidder  |
| **Safeframes OK**         | Yes                 | **Supports Deals**        | No                 |
| **Floors Module Support** | Yes                 | **First Party Data Support**| No               |
| **User IDs**              | All                 | **ORTB Blocking Support** | No                 |
| **Privacy Sandbox**       | Check with bidder   | **Prebid Server App Support** | Yes            |

## "Send All Bids" Ad Server Keys

These are the bidder-specific keys that would be targeted within GAM in a Send-All-Bids scenario. GAM truncates keys to 20 characters.

| Bidder Key                | GAM Key Description           | Example Value               |
|---------------------------|-------------------------------|-----------------------------|
| `hb_pb_pixfuture`         | Price bucket key             | e.g., `1.23`               |
| `hb_bidder_pixfuture`     | Bidder key                   | e.g., `pixfuture`          |
| `hb_adid_pixfuture`       | Ad ID key                    | e.g., `12345`              |
| `hb_size_pixfuture`       | Ad size key                  | e.g., `300x250`            |
| `hb_source_pixfuture`     | Source key                   | e.g., `client`             |
| `hb_format_pixfuture`     | Format key                   | e.g., `banner`             |
| `hb_cache_host_pixfuture` | Cache host key               | e.g., `cache.host.com`     |
| `hb_cache_id_pixfuture`   | Cache ID key                 | e.g., `abcdef`             |
| `hb_uuid_pixfuture`       | UUID key                     | e.g., `123e4567-e89b-12d3` |
| `hb_cache_path_pixfuture` | Cache path key               | e.g., `/cache`             |
| `hb_deal_pixfuture`       | Deal ID key                  | e.g., `deal123`            |

## Bid Params

| Name         | Scope     | Description                    | Example                        | Type   |
|--------------|-----------|--------------------------------|--------------------------------|--------|
| `pix_id`     | Required  | Placement ID                   | `12345`                        | String |


```