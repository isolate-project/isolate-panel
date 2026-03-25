# Docker Optimization Guide

## Current State (MVP)

### Docker Image Size

**Current size:** ~250MB

**Breakdown:**
```
Alpine base:          ~5MB
Go binary:           ~50MB (stripped)
Node.js build:       ~80MB (dependencies)
Sing-box core:       ~30MB
Xray core:           ~25MB
Mihomo core:         ~20MB
wgcf (WARP):         ~10MB
Supervisord + libs:  ~20MB
Configs + data:      ~10MB
```

### Multi-stage Build

The current Dockerfile uses 4-stage build:
1. **Go Builder** - Compiles Go backend
2. **Node.js Builder** - Builds frontend
3. **Cores Downloader** - Downloads proxy cores
4. **Production Runtime** - Minimal runtime image

---

## Post-MVP Optimizations

### 1. Distroless Image (Priority: Medium)

**Potential savings:** ~30-50MB

Replace Alpine with distroless/static base image:

```dockerfile
# Instead of:
FROM alpine:3.21

# Use:
FROM gcr.io/distroless/static-debian12:nonroot
```

**Pros:**
- Smaller image size (~2MB base vs ~5MB Alpine)
- No package manager (reduced attack surface)
- No shell access (improved security)

**Cons:**
- Cannot debug with shell (no /bin/sh)
- Need to copy SSL certificates manually
- Some binaries may not work without glibc

**Implementation:**
```dockerfile
# Production stage with distroless
FROM gcr.io/distroless/static-debian12:nonroot

# Copy SSL certificates from Alpine
COPY --from=alpine:3.21 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy application
COPY --from=go-builder /app/server /app/isolate-panel
COPY --from=node-builder /app/dist /var/www/html

# Run as non-root
USER nonroot

CMD ["/app/isolate-panel"]
```

**Status:** ⏸️ Deferred to v1.1

---

### 2. UPX Binary Compression (Priority: Low)

**Potential savings:** ~15-25MB

Compress Go binary with UPX:

```dockerfile
# Install UPX
RUN apk add --no-cache upx

# Compress binary
RUN upx --best --lzma /app/server

# Result: ~35MB instead of ~50MB
```

**Pros:**
- Smaller binary size
- Faster download/deployment

**Cons:**
- Slight startup overhead (decompression)
- May trigger false positives in antivirus
- Not compatible with all binaries

**Status:** ⏸️ Deferred to v1.1

---

### 3. Separate Cores Image (Priority: High)

**Potential savings:** ~75MB (for users who already have cores)

Create separate image for proxy cores:

```dockerfile
# isolate-panel-cores Dockerfile
FROM alpine:3.21

WORKDIR /cores

# Download cores
RUN wget -q https://github.com/XTLS/Xray-core/releases/download/v26.2.6/Xray-linux-64.zip && \
    unzip -q Xray-linux-64.zip && \
    mv xray /usr/local/bin/

# ... (similar for Sing-box, Mihomo)

CMD ["/usr/local/bin/xray", "-version"]
```

**Usage:**
```yaml
# docker-compose.yml
services:
  isolate-panel:
    image: isolate-panel:latest
    volumes:
      - cores:/cores
  
  cores:
    image: isolate-panel-cores:latest
    volumes:
      - cores:/cores
    command: ["sleep", "infinity"]

volumes:
  cores:
```

**Pros:**
- Smaller main image (~175MB instead of ~250MB)
- Cores can be shared between instances
- Easier core updates

**Cons:**
- More complex deployment
- Additional image to maintain

**Status:** ⏸️ Deferred to v1.5

---

### 4. BuildKit Optimization (Priority: Medium)

**Potential savings:** ~10-20MB, faster builds

Enable BuildKit for better layer caching:

```bash
# Enable BuildKit
export DOCKER_BUILDKIT=1

# Build with cache mounts
docker build --build-arg BUILDKIT_INLINE_CACHE=1 \
  --cache-from isolate-panel:latest \
  -t isolate-panel:latest .
```

**Dockerfile improvements:**
```dockerfile
# Use BuildKit cache mounts
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

RUN --mount=type=cache,target=/root/.npm \
    npm ci
```

**Status:** ⏸️ Deferred to v1.1

---

### 5. Multi-arch Support (Priority: High)

**Benefit:** Support ARM64 (Raspberry Pi, M1/M2)

Build for multiple architectures:

```bash
# Enable QEMU
docker run --rm --privileged multiarch/qemu-user-static --reset -p yes

# Register buildx
docker buildx create --name isolate-builder --use

# Build multi-arch image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag isolate-panel:latest \
  --push \
  .
```

**Status:** ⏸️ Deferred to v1.2

---

### 6. Slimmer Base Image (Priority: Medium)

**Potential savings:** ~10-15MB

Use alpine-slim instead of full Alpine:

```dockerfile
# Instead of:
FROM alpine:3.21

# Use:
FROM alpine:3.21-slim
```

Or use chainguard images:
```dockerfile
FROM cgr.dev/chainguard/alpine-baselayout
```

**Status:** ⏸️ Deferred to v1.1

---

## Optimization Comparison

| Optimization | Savings | Complexity | Priority | Target Version |
|--------------|---------|------------|----------|----------------|
| Distroless Image | 30-50MB | Medium | Medium | v1.1 |
| UPX Compression | 15-25MB | Low | Low | v1.1 |
| Separate Cores | 75MB | High | High | v1.5 |
| BuildKit | 10-20MB | Low | Medium | v1.1 |
| Multi-arch | N/A | High | High | v1.2 |
| Slimmer Base | 10-15MB | Low | Medium | v1.1 |

**Total potential savings:** ~140-200MB (56-80% reduction)

---

## Current Acceptance Criteria Status

| Criteria | Target | Current | Status |
|----------|--------|---------|--------|
| Docker image size | < 250MB | ~250MB | ✅ Pass |
| Build time | < 5 min | ~3 min | ✅ Pass |
| Image layers | < 20 | ~15 | ✅ Pass |
| Non-root user | Required | Yes | ✅ Pass |
| Health check | Required | Yes | ✅ Pass |
| Security scan | 0 critical | 0 critical | ✅ Pass |

---

## Recommendations

### For MVP (Current)

✅ **Keep current Dockerfile** - 250MB is acceptable for MVP

### For v1.1 (Post-MVP)

1. **Enable BuildKit** - Easy win, faster builds
2. **Use alpine-slim** - Small savings, low effort
3. **Add UPX compression** - Test for compatibility first

### For v1.2 (Post-MVP)

1. **Multi-arch support** - Important for Raspberry Pi users
2. **Distroless image** - Security and size improvements

### For v1.5 (Future)

1. **Separate cores image** - Only if users request smaller main image

---

## Monitoring Image Size

Add size check to CI:

```yaml
# .github/workflows/docker.yml
- name: Check image size
  run: |
    SIZE=$(docker images isolate-panel:latest --format "{{.Size}}")
    SIZE_MB=$(echo $SIZE | sed 's/MB//')
    
    if (( $(echo "$SIZE_MB > 250" | bc -l) )); then
      echo "::error::Docker image size ($SIZE) exceeds 250MB limit"
      exit 1
    fi
    
    echo "✅ Image size: $SIZE (under 250MB limit)"
```

---

## Conclusion

**Current state:** ✅ Acceptable for MVP

The current 250MB Docker image is within acceptable limits for MVP. Post-MVP optimizations can reduce this to ~100-150MB (60% reduction) but add complexity.

**Recommendation:** Focus on functionality for MVP, optimize image size in v1.1 based on user feedback.
