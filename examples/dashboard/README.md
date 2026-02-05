# Example Dashboard

Generate an example dashboard from sample data:

```bash
confvis aggregate \
  -c ../../testdata/sample.json:60 \
  -c ../../testdata/sample_failing.json:40 \
  -o ./output
```

Then open `./output/dashboard/index.html` in a browser.

## Embedding in Confluence or Other Systems

### Option A: iframe embedding (easiest)

Host the dashboard somewhere (GitHub Pages, S3, internal server) and embed via iframe:

```html
<iframe src="https://your-host.com/dashboard/index.html" width="100%" height="600"></iframe>
```

In Confluence:
- Use the "HTML" macro or iframe macro
- Note: Confluence Cloud may require admin to whitelist iframe sources

### Option B: Fragment mode (more flexible)

Generate just the content without HTML wrapper using `--fragment`:

```bash
confvis aggregate \
  -c ../../testdata/sample.json:60 \
  -c ../../testdata/sample_failing.json:40 \
  -o ./output \
  --fragment
```

The output has no DOCTYPE - just `<style>` and `<div>` elements that can be:
- Pasted directly into Confluence HTML macro
- Embedded in other HTML pages
- Injected into existing dashboards

### Verifying fragment output

```bash
# Check that output starts with <style>, not <!DOCTYPE
head -5 ./output/dashboard/index.html
```
