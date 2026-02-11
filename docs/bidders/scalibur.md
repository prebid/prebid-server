# Scalibur

The Scalibur adapter supports Banner and Video media types via OpenRTB 2.6.

## Registration
Contact [support@scalibur.io](mailto:support@scalibur.io) to obtain a valid `placementId`.

## Bid Params
| Name | Type | Description | Notes                 |
| :--- | :--- | :--- |:----------------------|
| `placementId` | string | **Required**. Scalibur placement identifier. | Example: `"468acd11"` |
| `bidfloor` | number | Optional. Minimum bid floor price. |                       |
| `bidfloorcur` | string | Optional. Currency for the bid floor. | Default is `USD`.     |

## User Sync
Supports iframe synchronization. Prebid Server handles the mapping of the sync ID to the `user.buyeruid` field in outgoing OpenRTB requests.