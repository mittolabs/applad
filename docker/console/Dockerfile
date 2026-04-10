FROM ghcr.io/cirruslabs/flutter:stable AS builder
WORKDIR /app

# Copy SDK dependency first for layer caching
COPY sdks/dart/ ./sdks/dart/

# Copy console and resolve dependencies
COPY console/pubspec.yaml console/pubspec.lock* ./console/
RUN cd console && flutter pub get

# Build
COPY console/ ./console/
RUN cd console && flutter build web --release --dart2js-optimization=O2

FROM nginx:alpine
COPY --from=builder /app/console/build/web /usr/share/nginx/html
COPY docker/console/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
