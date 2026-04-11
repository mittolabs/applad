FROM ghcr.io/cirruslabs/flutter:stable AS builder
WORKDIR /app

# Copy SDK dependency first for layer caching
COPY sdks/dart/ ./sdks/dart/

# Copy console and resolve dependencies
COPY console/pubspec.yaml console/pubspec.lock* ./console/
RUN cd console && flutter pub get

# Build
COPY console/ ./console/
# O1 produces smaller memory footprint than O2 during compilation while
# still tree-shaking dead code. Use --no-tree-shake-icons to avoid the
# extra icon analysis pass that peaks memory usage on low-RAM hosts.
# Set old_gen_heap_size to give the Dart VM a predictable upper bound
# instead of grabbing all available RAM.
RUN cd console && DART_VM_OPTIONS="--old_gen_heap_size=1500" \
    flutter build web --release \
    --dart2js-optimization=O1 \
    --no-tree-shake-icons

FROM nginx:alpine
COPY --from=builder /app/console/build/web /usr/share/nginx/html
COPY docker/console/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
