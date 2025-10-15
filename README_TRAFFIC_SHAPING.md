# Traffic Shaping Testing - Complete Setup ✅

## 🎉 All Test Files Created Successfully!

You now have everything you need to test your traffic shaping module implementation.

## 📦 Files Created

### 1. Test Auction Requests
- **`test_auction.json`** - Simple single-bidder test (original)
- **`test_auction_multi_bidder.json`** - Multi-bidder test with 5 bidders across 3 ad slots ⭐

### 2. Traffic Shaping Configuration
- **`traffic_shaping_config.json`** - Sample config matching your RTD format
- **`traffic_shaping_test_scenarios.json`** - 8 detailed test scenarios

### 3. Documentation
- **`TRAFFIC_SHAPING_TEST_GUIDE.md`** - Comprehensive testing guide
- **`TRAFFIC_SHAPING_SUMMARY.md`** - Quick reference summary
- **`README_TRAFFIC_SHAPING.md`** - This file

### 4. Test Scripts
- **`run.sh`** - PBS management script (start/stop/status/logs/test)
- **`test_traffic_shaping.sh`** - Automated traffic shaping tests ⭐

### 5. Helper Files
- **`PBS_LOCAL_GUIDE.md`** - General PBS local testing guide

---

## 🚀 Quick Start

### Step 1: Verify PBS is Running
```bash
./run.sh status
```

### Step 2: Run Baseline Test
```bash
./test_traffic_shaping.sh baseline
```

**Expected Output**:
```
✓ Found 4 active bidders:
  - appnexus
  - criteo
  - openx
  - pubmatic
```

### Step 3: Run Full Test Suite
```bash
./test_traffic_shaping.sh all
```

This runs 8 automated tests:
1. ✅ Baseline bidder count
2. ✅ Specific bidder check
3. ✅ Consistency test (5 runs)
4. ✅ Response time analysis
5. ✅ Warning detection
6. ✅ Debug mode test
7. ✅ Traffic shaping detection
8. ✅ Bidder comparison

---

## 📊 Your Traffic Shaping Format

Based on: https://rtd.mile.so/ts-static/0OsUhO/US/m-ios/safari/ts.json

```json
{
  "meta": {
    "createdAt": 1760345713
  },
  "response": {
    "schema": {
      "fields": ["gpID"]
    },
    "skipRate": 100,
    "userIdVendors": [
      "33acrossId", "criteoId", "hadronId", "idl_env", "index",
      "magnite", "medianet", "openx", "pubcid", "pubmatic",
      "sovrn", "tdid", "uid2"
    ],
    "values": {
      "ad-slot-identifier": {
        "bidderName": {
          "300x250": 1,  // 1 = allow, 0 = block
          "728x90": 0
        }
      }
    }
  }
}
```

### Key Fields Explained

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `skipRate` | 0-100 | Global traffic throttle percentage | `100` = block all, `0` = allow all |
| `userIdVendors` | array | Whitelist of allowed user ID providers | `["pubcid", "tdid", "uid2"]` |
| `values` | object | Per-slot bidder + size configuration | See below |
| `gpID` | string | Google Publisher ad slot identifier | `/homepage/banner` |
| `bidderName` | object | Bidder-specific size rules | `{"300x250": 1}` |
| `size` | 0 or 1 | Allow (1) or block (0) this size | `1` |

---

## 🧪 Test Scenarios

### Scenario 1: All Bidders Allowed ✅
```bash
# Edit traffic_shaping_config.json: skipRate=0, all bidders=1
./test_traffic_shaping.sh baseline
```
**Expected**: 4 bidders (appnexus, criteo, openx, pubmatic)

### Scenario 2: Block Specific Bidders
```json
{
  "values": {
    "/homepage/top-banner": {
      "appnexus": {"300x250": 1},
      "openx": {"300x250": 0},      // ❌ Blocked
      "pubmatic": {"300x250": 1},
      "criteo": {"300x250": 0}       // ❌ Blocked
    }
  }
}
```
**Expected**: 2 bidders (appnexus, pubmatic)

### Scenario 3: Global Skip Rate 100%
```json
{
  "skipRate": 100
}
```
**Expected**: 0 bidders (all blocked)

### Scenario 4: Size-Specific Filtering
```json
{
  "values": {
    "/homepage/top-banner": {
      "appnexus": {"300x250": 1, "728x90": 1},
      "openx": {"300x250": 1, "728x90": 0}  // ❌ Blocked for 728x90
    }
  }
}
```
**Expected**: 
- 300x250: appnexus, openx
- 728x90: appnexus only

---

## 🛠️ Test Commands

### Basic Tests
```bash
# Check which bidders are active
./test_traffic_shaping.sh baseline

# Test specific bidder
./test_traffic_shaping.sh bidder appnexus

# Check response times
./test_traffic_shaping.sh timing

# Run consistency check (10 times)
./test_traffic_shaping.sh consistency 10
```

### Manual Auction Test
```bash
# Run auction and see bidders
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d @test_auction_multi_bidder.json | \
  jq '{bidders: .ext.responsetimemillis | keys}'
```

### With Debug Mode
```bash
# Add debug to request
jq '.ext.prebid.debug = true' test_auction_multi_bidder.json | \
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d @- | jq '.ext.debug'
```

---

## 📋 Verification Checklist

When testing your traffic shaping module:

- [ ] **Baseline Test**: All 4 bidders respond without traffic shaping
- [ ] **Skip Rate 0**: All configured bidders respond
- [ ] **Skip Rate 100**: NO bidders respond
- [ ] **Skip Rate 50**: ~50% of requests have bidders
- [ ] **Bidder Blocking**: Specific bidders are filtered out
- [ ] **Size Filtering**: Bidders blocked for specific sizes only
- [ ] **User ID Filtering**: Only whitelisted user IDs passed to bidders
- [ ] **Unknown Slot**: Default behavior when gpID not in config
- [ ] **Logs**: Traffic shaping decisions logged
- [ ] **Performance**: No significant latency added

---

## 🔍 Debugging Tips

### 1. Check PBS Logs
```bash
./run.sh logs 50 follow
```

Look for:
- `[traffic_shaping]` entries
- Bidder filtering decisions
- Skip rate calculations

### 2. Enable Debug Mode
Add to auction request:
```json
{
  "ext": {
    "prebid": {
      "debug": true
    }
  }
}
```

### 3. Compare Before/After
```bash
# Before traffic shaping
curl -s ... | jq '.ext.responsetimemillis | keys' > before.txt

# After traffic shaping
curl -s ... | jq '.ext.responsetimemillis | keys' > after.txt

# Diff
diff before.txt after.txt
```

### 4. Test Individual Bidders
```bash
for bidder in appnexus criteo openx pubmatic; do
  echo -n "$bidder: "
  ./test_traffic_shaping.sh bidder $bidder | grep -q "responded" && echo "✓" || echo "✗"
done
```

---

## 📊 Current Setup Status

### ✅ Working
- PBS running on port 8000
- 4 bidders active: appnexus, criteo, openx, pubmatic
- Multi-bidder test auction configured
- Test scripts ready

### ⚠️ Disabled Bidders
- rubicon (disabled in PBS config)
- ix (disabled in PBS config)

To enable them, add to `pbs.yaml`:
```yaml
adapters:
  rubicon:
    disabled: false
  ix:
    disabled: false
```

---

## 🎯 Integration with Your Module

Your traffic shaping module should hook into PBS at these points:

1. **Load Config** (startup or periodic refresh)
   ```go
   config := loadTrafficShapingConfig(url)
   ```

2. **Process Request** (before bidder calls)
   ```go
   func filterBidders(req *openrtb2.BidRequest, config *TSConfig) []string {
       // Extract gpID from imp[].ext.gpid
       // Check skipRate
       // Filter bidders by size
       // Filter user IDs
       return allowedBidders
   }
   ```

3. **Return Filtered List**
   ```go
   // PBS calls only allowed bidders
   ```

---

## 📚 Documentation Files

| File | Purpose |
|------|---------|
| `TRAFFIC_SHAPING_TEST_GUIDE.md` | Comprehensive testing guide with all scenarios |
| `TRAFFIC_SHAPING_SUMMARY.md` | Quick reference and commands |
| `README_TRAFFIC_SHAPING.md` | This file - complete setup overview |
| `PBS_LOCAL_GUIDE.md` | General PBS local testing guide |

---

## 🚀 Next Steps

1. ✅ Test files created
2. ✅ PBS running locally
3. ✅ Baseline test passing (4 bidders)
4. ⏳ Implement traffic shaping module
5. ⏳ Configure PBS to load module
6. ⏳ Run test scenarios
7. ⏳ Verify filtering behavior
8. ⏳ Performance testing

---

## 💡 Example Test Flow

```bash
# 1. Start PBS
./run.sh start

# 2. Baseline test (no traffic shaping)
./test_traffic_shaping.sh baseline
# Output: 4 bidders

# 3. Enable traffic shaping module
# (Configure your module to load traffic_shaping_config.json)

# 4. Test with traffic shaping
./test_traffic_shaping.sh all
# Should show filtered bidders

# 5. Test skip rate
# Edit traffic_shaping_config.json: "skipRate": 100
./run.sh restart
./test_traffic_shaping.sh baseline
# Output: 0 bidders

# 6. Test selective blocking
# Edit config: Block openx and criteo
./run.sh restart
./test_traffic_shaping.sh baseline
# Output: 2 bidders (appnexus, pubmatic)
```

---

## 🎉 You're All Set!

Everything is ready for testing your traffic shaping module. Run the tests, verify the behavior, and iterate on your implementation.

**Quick Test**: `./test_traffic_shaping.sh all`

Good luck! 🚀

