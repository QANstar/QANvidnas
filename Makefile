.PHONY: dev build docker apk-tv apk-mobile clean

# Build React frontend
build-client:
	cd client && npm ci && npm run build

# Copy client build to server for embedding
embed-client: build-client
	mkdir -p server/internal/router/web/dist
	cp -r client/dist/* server/internal/router/web/dist/

# Build Go server binary
build-server: embed-client
	cd server && go build -o ../qanvidnas .

# Build both (with embed)
build: build-server

# Development mode - start server only
dev:
	cd server && go run .

# Dependencies
deps:
	cd server && go mod tidy
	cd client && npm ci

# Docker build
docker:
	docker build -t qanvidnas:latest -f Dockerfile .

# Build Android TV APK
apk-tv:
	cd client && npm run build
	cd client && npx cap sync android
	cd client/android && ./gradlew assembleRelease
	@echo "TV APK: client/android/app/build/outputs/apk/release/app-release.apk"

# Build Android Mobile APK
apk-mobile:
	cd client && npm run build
	cd client && npx cap sync android
	cd client/android && ./gradlew assembleRelease
	@echo "Mobile APK: client/android/app/build/outputs/apk/release/app-release.apk"

# Clean build artifacts
clean:
	rm -rf server/internal/router/web/dist
	rm -rf client/dist
	rm -rf client/android/app/build
	rm -f qanvidnas
