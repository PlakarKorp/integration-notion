# 📦 integration-notion

`integration-notion` is a Plakar plugin that lets you back up your Notion pages or workspace directly into a Plakar repository.

> 🔐 All content is fetched via the Notion API and saved as structured JSON files, including both data and metadata.

---

## ⚙️ Requirements

- [**Plakar**](https://github.com/politaire/plakar) installed with plugin support
- A valid [**Notion API token**](https://www.notion.com/my-integrations) (`ntn_xxx`)  
- The target Notion pages must be **shared** with the integration you created
---

## 📦 Installation

This plugin is distributed via Plakar’s internal plugin system. Assuming you already have Plakar installed:

```bash
plakar pkg install integration-notion
```

That’s it — you’re ready to configure and use it.

---

## 🧪 Configuration

Create or edit your Plakar repository’s `config.yml` file and add a new remote under `remotes:` that points to Notion:

```yaml
remotes:
  my-notion-backup:
    location: 'notion://'
    token: 'ntn_<your-notion-api-token>'
```

Make sure your token is valid and that the integration has access to the pages you want to back up.

---

## 🚀 Usage

Once your config is in place, you can run the following command:

```bash
./plakar -a <repo> backup @<config-name>
```

- Replace `<repo>` with the path to your Plakar repository
- Replace `<config-name>` with the name you used in your `config.yml` (e.g. `my-notion-backup`)

This will connect to Notion and back up all accessible content into your Plakar repo.

---

## 📂 Backup Format

Backed-up content is stored as **JSON files**, including:
- Page content
- Metadata (title, ID, parent, etc.)
- Block structure and types

This ensures that future restoration (or data reuse) is flexible and complete.

---

## 🔄 Restoration

> 🧪 Restoration is **not available yet**, but support is planned in a future release.

Stay tuned!

---

## 🛠️ Tips

- **Sharing:** Make sure your integration is shared with each Notion page you want to back up.  
  → Create an [Integration in Notion](https://www.notion.com/my-integrations) and share it with the pages you want to back up.

  → See [Notion’s guide on integrations](https://developers.notion.com/docs/getting-started#step-1-create-an-integration) for how to create and share your token properly.
- **Security:** Keep your token safe — don’t commit it into version control.
- **Selective backups:** Currently, the plugin pulls all shared pages — filtering support may come later.

---

## ❓ FAQ / Troubleshooting

**Q:** I get an error saying access is denied.  
**A:** Check that your integration is shared with the page in Notion. Even with a valid token, it can’t access pages that aren’t shared.

---

## 📬 Feedback

Found a bug? Have a feature request? Open an issue or ping the Plakar team.
