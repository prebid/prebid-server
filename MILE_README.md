# MILE Modules - Complete Documentation

This guide serves as the central documentation for MILE modules in Prebid Server, covering functionality, configuration, deployment, and testing.

## Table of Contents

- [Modules Overview](#modules-overview)
  - [Traffic Shaping](#1-traffic-shaping-module)
  - [Common Module (Geo/Device)](#2-common-module)
  - [Floors Module](#3-floors-module)
- [Quick Start](#quick-start)
- [Deployment & IP Resolution](#deployment--ip-resolution)
  - [Local Development](#scenario-1-local-development-runsh)
  - [Local Docker Build](#scenario-2-local-docker-build)
  - [GitHub Actions CI/CD](#scenario-3-github-actions-cicd)
- [Configuration Guide (pbs.yaml)](#configuration-guide-pbsyaml)
  - [Configuration Parameters](#configuration-parameters)
  - [Hook Execution Plan](#hook-execution-plan)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [File Structure](#file-structure)

---

## Modules Overview

### 1. Traffic Shaping Module

**Path**: `modules/mile/trafficshaping/`

Allows publishers to dynamically control which bidders and ad sizes are allowed for specific placements based on a remote configuration.

**Key Features**:
- **GPID-based Shaping**: Filter bidders and sizes per Global Placement ID.
- **Dynamic URL Construction**: Config URLs built using `siteID`, `country`, `device`, `browser`.
- **Skip Rate Gating**: Deterministic sampling to skip shaping.
- **Fail-open Behavior**: Auctions proceed normally if config fetch fails.
- **User ID Filtering**: Prune `user.ext.eids` to allowed vendors.

**Configuration Example**:
```yaml
hooks:
  enabled: true
  modules:
    mile:
      trafficshaping:
        enabled: true
        base_endpoint: "https://example.com/ts-server/"
        geo_db_path: "tmp/GeoLite2-Country.mmdb" # For IP fallback
```

### 2. Common Module

**Path**: `modules/mile/common/`

Provides shared functionality for resolving request context (Geo, Device, Browser).

**Key Features**:
- **Geo Resolution**: `device.geo.country` OR IP-based fallback (MaxMind/HTTP).
- **Device Detection**: Maps OpenRTB types to `mobile`, `tablet`, `desktop`.
- **Browser Detection**: Parses User-Agent string.

### 3. Floors Module

**Path**: `modules/mile/floors/`

Manages floor prices for auctions. (See module source for specific configurations).

---

## Quick Start

### Prerequisites
- Go 1.22+
- Docker (optional, for containerized deployment)
- MaxMind Account ID & License Key (free from [MaxMind](https://www.maxmind.com/en/geolite2/signup))
  - **Account ID**: Found in your MaxMind account dashboard
  - **License Key**: Generated from "My License Key" section

### Local Setup (5 Minutes)

1. **Clone & Config**:
   ```bash
   git clone <repo>
   cp .env.example .env
   # Add MAXMIND_ACCOUNT_ID and MAXMIND_LICENSE_KEY to .env
   ```

2. **Start Server**:
   ```bash
   ./run.sh start
   ```
   *Note: This automatically downloads the MaxMind database if missing.*

3. **Verify**:
   ```bash
   ./run.sh status
   ./run.sh test
   ```

---

## Deployment & IP Resolution

MILE modules support IP-based geo resolution using **MaxMind GeoLite2-Country**. The database is **automatically downloaded** when missing in all deployment scenarios.

### Scenario 1: Local Development (run.sh)

**Setup**:
1. Create `.env` file:
   ```bash
   cp .env.example .env
   ```

2. Add credentials to `.env`:
   ```bash
   MAXMIND_ACCOUNT_ID=your_account_id_here
   MAXMIND_LICENSE_KEY=your_license_key_here
   ```

3. Configure `pbs.yaml` for local dev:
   ```yaml
   hooks:
     modules:
       mile:
         trafficshaping:
           geo_db_path: "tmp/GeoLite2-Country.mmdb"  # Relative path
   ```

**Running**:
```bash
./run.sh start
```

**What Happens**:
- Script checks if `tmp/GeoLite2-Country.mmdb` exists
- If missing, sources `.env` and downloads database automatically
- Creates `tmp/` directory if needed
- Builds and starts Prebid Server

**Verify**:
```bash
# Check database was downloaded
ls -lh tmp/GeoLite2-Country.mmdb

# Check server status
./run.sh status

# Test auction endpoint
./run.sh test
```

**Without MaxMind**:
- Server starts normally
- IP resolution won't work (requests must include `device.geo.country`)
- Warning message displayed

---

### Scenario 2: Local Docker Build

**Setup**:
1. Create `.env` file (if not already created):
   ```bash
   cp .env.example .env
   # Add MAXMIND_ACCOUNT_ID and MAXMIND_LICENSE_KEY
   ```

2. Configure `pbs.yaml` for Docker:
   ```yaml
   hooks:
     modules:
       mile:
         trafficshaping:
           geo_db_path: "/opt/maxmind/GeoLite2-Country.mmdb"  # Absolute path
   ```

**Build**:
```bash
# Option 1: Source .env and use environment variables
source .env
docker build \
  --build-arg MAXMIND_ACCOUNT_ID="$MAXMIND_ACCOUNT_ID" \
  --build-arg MAXMIND_LICENSE_KEY="$MAXMIND_LICENSE_KEY" \
  -t prebid-server .

# Option 2: Pass directly
docker build \
  --build-arg MAXMIND_ACCOUNT_ID="your_account_id" \
  --build-arg MAXMIND_LICENSE_KEY="your_license_key" \
  -t prebid-server .
```

**What Happens During Build**:
- Dockerfile checks if database exists in build stage
- If missing and credentials provided, downloads database
- Extracts `.mmdb` file from tar.gz archive
- Copies to `/opt/maxmind/GeoLite2-Country.mmdb` in final image
- Sets proper permissions (readable by prebid user)

**Verify**:
```bash
# Check if database exists in image
docker run --rm prebid-server ls -lh /opt/maxmind/GeoLite2-Country.mmdb

# Run container
docker run -p 8000:8000 \
  -v $(pwd)/pbs.yaml:/usr/local/bin/pbs.yaml \
  prebid-server
```

**Build Without MaxMind**:
- Build succeeds (database download skipped)
- Image created without database
- Server runs but IP resolution won't work

---

### Scenario 3: GitHub Actions CI/CD

**Setup Secrets**:
1. Go to GitHub Repository:
   - Navigate to: **Settings → Secrets and variables → Actions**

2. Add Secrets:
   - **Name**: `MAXMIND_ACCOUNT_ID`
     - **Value**: Your MaxMind account ID
   - **Name**: `MAXMIND_LICENSE_KEY`
     - **Value**: Your MaxMind license key
   - Click "Add secret" for each

**Workflow Configuration**:
The workflow (`.github/workflows/release.yml`) automatically passes secrets as build args:

```yaml
- name: Build image
  run: |
    docker build \
      --build-arg MAXMIND_ACCOUNT_ID="${{ secrets.MAXMIND_ACCOUNT_ID }}" \
      --build-arg MAXMIND_LICENSE_KEY="${{ secrets.MAXMIND_LICENSE_KEY }}" \
      -t docker.io/prebid/prebid-server:${{ needs.publish-tag.outputs.releaseTag }} .
```

**Build Process**:
1. GitHub Actions runs workflow
2. Checks out code
3. Runs `docker build` with both secrets as build args
4. Database downloads during build if missing
5. Image pushed to Docker Hub with database included

**Verify in CI/CD**:
- Check build logs for "Downloading MaxMind database" message
- Verify image contains `/opt/maxmind/GeoLite2-Country.mmdb`
- Test pulled image locally

**Without Secrets**:
- Build succeeds (database download skipped)
- Image created without database
- Users must provide `device.geo.country` in requests

---

---

## Configuration Guide (pbs.yaml)

The `pbs.yaml` file is the main configuration file for Prebid Server. Here's a complete example with MILE modules configured:

### Complete Example

```yaml
host: ""
port: 8000

hooks:
  enabled: true
  modules:
    mile:
      trafficshaping:
        enabled: true
        base_endpoint: "http://localhost:8080/ts-server/"
        
        # IP-based geo resolution (choose one):
        # Option 1: MaxMind database (recommended - fastest)
        geo_db_path: "tmp/GeoLite2-Country.mmdb"  # Relative path for local dev
        # For Docker: geo_db_path: "/opt/maxmind/GeoLite2-Country.mmdb"
        
        # Option 2: HTTP geo service (alternative)
        # geo_lookup_endpoint: "http://geo-service.com/{ip}"
        # geo_cache_ttl_ms: 300000  # 5 minutes cache (only for HTTP resolver)
        
        refresh_ms: 30000           # Config refresh interval (ms)
        request_timeout_ms: 2000     # HTTP request timeout (ms)
        prune_user_ids: false       # Enable user ID vendor filtering
        sample_salt: "pbs"          # Salt for deterministic sampling
  
  default_account_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          processed_auction_request:
            groups:
              - timeout: 50
                hook_sequence:
                  - module_code: mile.trafficshaping
                    hook_impl_code: default

gdpr:
  host_vendor_id: 0
  default_value: "0"

analytics:
  adapters:
    "*":
      enabled: true

account_defaults:
  debug_allow: true
```

### Configuration Parameters

#### Traffic Shaping Module

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | - | Enable/disable the module |
| `base_endpoint` | string | Yes* | - | Base URL for dynamic config fetching (must end with `/`) |
| `endpoint` | string | Yes* | - | Static config URL (legacy mode, use `base_endpoint` instead) |
| `geo_db_path` | string | No** | - | Path to MaxMind GeoIP2 database file |
| `geo_lookup_endpoint` | string | No** | - | HTTP endpoint for IP-based geo lookup (supports `{ip}` placeholder) |
| `geo_cache_ttl_ms` | integer | No | 300000 | TTL for geo lookup cache in ms (only for HTTP resolver) |
| `refresh_ms` | integer | No | 30000 | Config refresh interval in ms (min: 1000) |
| `request_timeout_ms` | integer | No | 1000 | HTTP request timeout in ms (min: 100) |
| `prune_user_ids` | boolean | No | false | Enable user ID vendor filtering |
| `sample_salt` | string | No | "pbs" | Salt for deterministic sampling |
| `allowed_countries` | array | No | [] | List of allowed countries (ISO 3166-1 alpha-2) |

> **\*** At least one of `base_endpoint` (dynamic mode) or `endpoint` (static mode) is required.  
> **\*\*** At least one of `geo_db_path` or `geo_lookup_endpoint` is required for IP-based geo resolution.

#### Geo Resolution Options

**Option 1: MaxMind Database (Recommended)**
```yaml
geo_db_path: "tmp/GeoLite2-Country.mmdb"  # Local dev (relative path)
# OR
geo_db_path: "/opt/maxmind/GeoLite2-Country.mmdb"  # Docker/Production (absolute path)
```

**Option 2: HTTP Geo Service**
```yaml
geo_lookup_endpoint: "http://geo-service.com/{ip}"
geo_cache_ttl_ms: 300000  # 5 minutes cache
```

**Note**: If both are configured, MaxMind (`geo_db_path`) takes priority.

### Environment-Specific Paths

| Environment | `geo_db_path` Value |
|-------------|---------------------|
| Local Development | `"tmp/GeoLite2-Country.mmdb"` (relative) |
| Docker/Production | `"/opt/maxmind/GeoLite2-Country.mmdb"` (absolute) |

### Hook Execution Plan

The `default_account_execution_plan` section defines when and how the module runs:

```yaml
default_account_execution_plan:
  endpoints:
    /openrtb2/auction:           # Endpoint to hook into
      stages:
        processed_auction_request:  # Stage: after request processing
          groups:
            - timeout: 50           # Max execution time (ms)
              hook_sequence:
                - module_code: mile.trafficshaping
                  hook_impl_code: default
```

**Available Endpoints**:
- `/openrtb2/auction` - Main auction endpoint
- `/openrtb2/amp` - AMP endpoint (optional)
- `/openrtb2/video` - Video endpoint (optional)

### Configuration Validation

Prebid Server validates the configuration on startup. Common errors:

| Error | Cause | Fix |
|-------|-------|-----|
| Missing `base_endpoint` | Required for dynamic mode | Add `base_endpoint` or `endpoint` |
| Invalid `geo_db_path` | File doesn't exist | Check path, run `./run.sh start` to download |
| Invalid `refresh_ms` | Value < 1000 | Set to >= 1000 |
| Invalid `request_timeout_ms` | Value < 100 | Set to >= 100 |
| Module not executing | Missing hook plan | Add `default_account_execution_plan` |

### Verifying Configuration

```bash
# Restart server to load new config
./run.sh restart

# Check logs for configuration errors
./run.sh logs 50

# Test with a request
./run.sh test
```

---

## Testing

### Traffic Shaping Tests

We provide comprehensive test scripts in the root directory.

**Run All Tests**:
```bash
./test_traffic_shaping.sh all
```

**Scenarios Covered**:
1. Baseline (all bidders active)
2. Bidder Blocking
3. Consistency Checks
4. Response Timing
5. Debug Mode

**Manual Test**:
```bash
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d @test_auction_multi_bidder.json | jq .
```

---

## Troubleshooting

### Common Issues

| Issue | Symptoms | Solution |
|-------|----------|----------|
| **Database Not Downloading** | "MAXMIND_LICENSE_KEY not set" warning | Check `.env` exists with `MAXMIND_ACCOUNT_ID` and `MAXMIND_LICENSE_KEY` |
| **IP Resolution Not Working** | Country not resolved from IP | Ensure `device.ip` or `device.ipv6` in request; verify `geo_db_path` |
| **Docker Build Fails** | MaxMind download error | Verify build args passed correctly; check license key validity |
| **Module Not Running** | No shaping applied | Check `hooks.enabled: true` and `default_account_execution_plan` |
| **Config Fetch Fails** | "fetch_failed" in analytics | Verify `base_endpoint` URL is accessible |

### Debugging Commands

```bash
# View recent logs
./run.sh logs 100 follow

# Check server status
./run.sh status

# Manual database download
./scripts/download-maxmind.sh tmp/GeoLite2-Country.mmdb

# Verify database file
ls -lh tmp/GeoLite2-Country.mmdb

# Test auction endpoint
curl -s http://localhost:8000/openrtb2/auction -X POST \
  -H "Content-Type: application/json" \
  -d '{"id":"test","imp":[{"id":"1"}]}' | jq .
```

---

## File Structure

```
prebid-server/
├── MILE_README.md               # This documentation
├── pbs.yaml                     # Main Prebid Server configuration
├── run.sh                       # Local development script
├── .env.example                 # Credentials template
├── .env                         # Local credentials (gitignored)
├── tmp/
│   └── GeoLite2-Country.mmdb    # MaxMind database (local dev, gitignored)
├── scripts/
│   └── download-maxmind.sh      # MaxMind database downloader
├── modules/mile/
│   ├── common/                  # Shared geo/device/browser resolution
│   ├── trafficshaping/          # Traffic shaping module
│   └── floors/                  # Floor price module
├── .github/workflows/
│   ├── release.yml              # Docker build & publish
│   └── publishonly.yml          # Publish only workflow
└── Dockerfile                   # Container build (includes MaxMind download)
```
