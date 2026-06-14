## Watchlist Addition Request

<!-- Use this template when requesting to add a new manga series to the CDC tracking watchlist.yaml file. -->

### Required Entry Details
Please provide the YAML configuration block you added to `watchlist.yaml`:

```yaml
- source: "[mangadex | mangafire | asurascans | mangaplus | mangatown | mangapill]"
  source_id: "[Internal ID from the source]"
  title: "[Full Series Title]"
  source_url: "[Full URL to the series page]"
```

### Rationale
<!-- Why should this series be tracked by the CDC pipeline? -->
<!-- e.g. It is highly popular, recently serialized, or heavily requested. -->

### Verification
- [ ] I have verified that the `source` is one of the supported providers (`mangadex`, `mangafire`, `asurascans`, `mangaplus`, `mangatown`, `mangapill`).
- [ ] I have verified that `source_id` correctly identifies the manga on the source platform.
- [ ] I have verified that the `source_url` is accessible and not behind a permanent captcha.
- [ ] I have ensured this series is not already tracked in the existing `watchlist.yaml`.
