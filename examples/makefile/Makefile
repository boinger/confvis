# Example Makefile targets for confvis integration
#
# Add these targets to your project's Makefile

.PHONY: confidence confidence-check confidence-dark

# Generate badge and dashboard
confidence:
	confvis generate -c confidence.json -o ./badges

# Generate in dark mode
confidence-dark:
	confvis generate -c confidence.json -o ./badges --dark

# Validate confidence.json without generating output
# Useful in CI to catch config errors early
confidence-check:
	confvis gauge -c confidence.json -o /dev/null
	@echo "Confidence config is valid"

# Generate confidence.json from test coverage
# Then generate the badge
.PHONY: update-confidence
update-confidence:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./... > /dev/null 2>&1 || true
	@COVERAGE=$$(go tool cover -func=coverage.out 2>/dev/null | grep total | awk '{print int($$3)}' || echo 0); \
	echo "Coverage: $$COVERAGE%"; \
	echo '{"title":"Code Quality","score":'"$$COVERAGE"',"threshold":75,"factors":[{"name":"Test Coverage","score":'"$$COVERAGE"',"weight":100}]}' > confidence.json
	@$(MAKE) confidence
	@echo "Badge updated"
