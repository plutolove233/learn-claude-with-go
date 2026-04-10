#!/usr/bin/env bash
# scripts/build.sh — 跨平台编译 + 版本管理
# 用法：
#   ./scripts/build.sh               # 编译当前版本
#   ./scripts/build.sh --bump patch  # 升 patch 版本后编译
#   ./scripts/build.sh --bump minor  # 升 minor 版本后编译
#   ./scripts/build.sh --bump major  # 升 major 版本后编译
#   VERSION=1.2.3 ./scripts/build.sh # 指定版本（CI/CD 场景）

set -euo pipefail

# ──────────────────────────────────────────────
# 配置
# ──────────────────────────────────────────────
BINARY_NAME="claudego"
VERSION_FILE="VERSION"
OUTPUT_DIR="dist"
MODULE=$(go list -m 2>/dev/null || echo "claudego")

# 编译目标平台列表：GOOS/GOARCH
TARGETS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/arm64"
)

# ──────────────────────────────────────────────
# 版本管理
# ──────────────────────────────────────────────
bump_version() {
  local current="$1"
  local part="$2"       # major | minor | patch

  IFS='.' read -r major minor patch <<< "$current"
  case "$part" in
    major) major=$((major + 1)); minor=0; patch=0 ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    patch) patch=$((patch + 1)) ;;
    *)
      echo "错误：--bump 参数须为 major / minor / patch" >&2
      exit 1
      ;;
  esac
  echo "${major}.${minor}.${patch}"
}

# 优先使用环境变量 VERSION（CI/CD 注入）
if [[ -n "${VERSION:-}" ]]; then
  CURRENT_VERSION="$VERSION"
else
  [[ -f "$VERSION_FILE" ]] || { echo "找不到 $VERSION_FILE，请先创建"; exit 1; }
  CURRENT_VERSION=$(tr -d '[:space:]' < "$VERSION_FILE")
fi

# 处理 --bump 参数
BUMP=""
for arg in "$@"; do
  case "$arg" in
    --bump) BUMP_NEXT=true ;;
    major|minor|patch) [[ "${BUMP_NEXT:-}" == "true" ]] && BUMP="$arg" ;;
  esac
done

if [[ -n "$BUMP" ]]; then
  NEW_VERSION=$(bump_version "$CURRENT_VERSION" "$BUMP")
  echo "版本升级：$CURRENT_VERSION → $NEW_VERSION"
  echo "$NEW_VERSION" > "$VERSION_FILE"
  CURRENT_VERSION="$NEW_VERSION"
fi

VERSION_TAG="v${CURRENT_VERSION}"
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "───────────────────────────────────────"
echo " Binary  : $BINARY_NAME"
echo " Version : $VERSION_TAG"
echo " Commit  : $GIT_COMMIT"
echo " Time    : $BUILD_TIME"
echo "───────────────────────────────────────"

# ──────────────────────────────────────────────
# 编译
# ──────────────────────────────────────────────
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# 将版本信息通过 ldflags 注入到二进制（需在代码里声明对应变量）
LDFLAGS="-s -w \
  -X '${MODULE}/internal/version.Version=${VERSION_TAG}' \
  -X '${MODULE}/internal/version.Commit=${GIT_COMMIT}' \
  -X '${MODULE}/internal/version.BuildTime=${BUILD_TIME}'"

SUCCESS=0
FAIL=0

for target in "${TARGETS[@]}"; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"

  EXT=""
  [[ "$GOOS" == "windows" ]] && EXT=".exe"

  OUT="${OUTPUT_DIR}/${BINARY_NAME}_${VERSION_TAG}_${GOOS}_${GOARCH}${EXT}"

  printf "  %-30s " "${GOOS}/${GOARCH}"

  if GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0  go build \
      -trimpath \
      -ldflags "$LDFLAGS" \
      -o "$OUT" \
      ./cmd/claudego/ 2>/tmp/build_err; then
    SIZE=$(du -sh "$OUT" | cut -f1)
    echo "✓  ${SIZE}"
    SUCCESS=$((SUCCESS + 1))
  else
    echo "✗  FAILED"
    cat /tmp/build_err
    FAIL=$((FAIL + 1))
  fi
done

echo "───────────────────────────────────────"
echo " 成功 $SUCCESS / 失败 $FAIL  →  ./${OUTPUT_DIR}/"
echo "───────────────────────────────────────"

# 生成 checksums
cd "$OUTPUT_DIR"
sha256sum ./* > "checksums_${VERSION_TAG}.txt" 2>/dev/null || \
  shasum -a 256 ./* > "checksums_${VERSION_TAG}.txt"
echo " Checksums: ${OUTPUT_DIR}/checksums_${VERSION_TAG}.txt"

[[ $FAIL -gt 0 ]] && exit 1
exit 0