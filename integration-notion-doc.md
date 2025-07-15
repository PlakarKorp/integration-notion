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

## 🚀 Usage

To back up your Notion pages, run:

```bash
plakar at /path/to/repo backup notion:// token=<ntn_xxx>
```

Suppsing you have a Plakar repository at `/path/to/repo` and Replace `<ntn_xxx>` with your actual Notion API token.

---

## 📂 Backup Format

Backed-up content is stored as **JSON files**, including:
- Page content
- Metadata (title, ID, parent, etc.)
- Block structure and types

---

## 🔄 Restoration

To restore your Notion backups, you can use the `restore` command:
considering you have a Plakar repository at `/path/to/repo` and and have a Notion page with the ID `<root_page_id>` where you want to restore the content:

```bash
plakar at /path/to/repo restore -to /path/to/restore notion:// token=<ntn_xxx> rootID=<root_page_id> <snapshot_id>
```

This will restore the backed-up Notion pages to the specified directory, maintaining the original structure.
/!\ Make sure the `root_page_id` corresponds to a valid Notion page where you have write access. Notion public API does not let you restore directly to a workspace.

---

## 🛠️ Tips

- **Sharing:** Make sure your integration is shared with each Notion page you want to back up.  
  → Create an [Integration in Notion](https://www.notion.com/my-integrations) and share it with the pages you want to back up.

  → See [Notion’s guide on integrations](https://developers.notion.com/docs/getting-started#step-1-create-an-integration) for how to create and share your token properly.
- **Security:** Keep your token safe — don’t commit it into version control.
- **Selective backups:** Currently, the plugin pulls all shared pages — filtering support may come later.

## 📬 Feedback

Found a bug? Have a feature request? Open an issue or ping the Plakar team.
