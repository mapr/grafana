# gdev provisioning bootstrap

Example `[provisioning] bootstrap_path` directory. At startup Grafana registers a **local**
repository named `bootstrap` over it and provisions everything declared here. No Git, no remote:
this is file provisioning, served by the same provisioning system that also backs Git Sync, so
the resources land as managed resources with a repository behind them.

```bash
cd devenv && ./setup.sh provisioning-bootstrap   # see what it changes, below
cd .. && make run
```

That command symlinks `devenv/provisioning-bootstrap.ini` to `conf/custom.ini` and links
`devenv/datasources.yaml`, which the dashboards here query. Linking the datasource file is not
scoped to `gdev-testdata`: on the next start Grafana deletes every datasource named in that
file's `deleteDatasources` list, overwrites any existing datasource whose name matches one it
declares, and makes `gdev-testdata` the instance default. `undev` unlinks the file but does not
undo any of that — those are database changes. Together with the library panel routing below,
that is the full set of instance-wide effects.

It also assumes the default org is org 1. File provisioning always creates `gdev-testdata` in
org 1, because `datasources.yaml` sets no `orgId`, while the bootstrap repository follows
`auto_assign_org_id`. Override that and the dashboards land in another org while their datasource
does not, so both panels fail to resolve it; setup refuses that combination when it can see it.

Then open **Administration → Provisioning**. The `bootstrap` repository is listed but its
configuration is read-only, because it comes from the `bootstrap_path` setting; the resources it
provisions are editable, and writing through the UI writes the files back here.

A bootstrap directory can also declare repositories and connections, which is the point of the
feature: an instance configures its own Git Sync from files on disk. This example stays on plain
resources, so it works with nothing external set up.

## What it declares

| Path                                                | Resource     | Why it is here                                    |
| --------------------------------------------------- | ------------ | ------------------------------------------------- |
| `playlist.json`                                     | Playlist     | Org-scoped, so it sits at the root                |
| `gdev-provisioning/_folder.json`                    | Folder       | Gives the directory a stable folder UID and title |
| `gdev-provisioning/random-walk.json`                | Dashboard    | Folder-scoped, uses `gdev-testdata`               |
| `gdev-provisioning/notes/_folder.json`              | Folder       | Nested folder, built by the same sync             |
| `gdev-provisioning/notes/slow-queries.json`         | Dashboard    | Dashboard below the nested folder                 |
| `gdev-provisioning/notes/library-panel-readme.json` | LibraryPanel | Folder-scoped, so it cannot live at the root      |

The playlist selects dashboards by the `provisioning` tag, which both dashboards carry.

## Notes

- A directory without a `_folder.json` still becomes a folder; the file is what pins its UID and
  title, so without it the UID is derived from the path.
- `bootstrap_path` is granted to the bootstrap repository implicitly — it does not need an entry
  in `[paths] permitted_provisioning_paths`.
- Clearing `bootstrap_path` deletes the repository **and everything it provisioned**. Run
  `./setup.sh undev`, restart, then run `./setup.sh undev` again: the first run only clears
  `bootstrap_path` (keeping the library panel's unified storage settings so the deletion reaches
  it), the second removes `conf/custom.ini`.
- The restart only _starts_ the deletion — finalizers do the work asynchronously. The second
  `undev` run therefore asks Grafana whether the repository is gone before unlinking, and keeps
  `conf/custom.ini` in place if it is not, or if Grafana cannot be reached. Removing the settings
  early strands the repository on a finalizer it can no longer satisfy: the library panel API is
  not served under the default configuration, so the delete keeps failing and the repository never
  goes away. Check progress with:

  ```bash
  curl -su admin:admin \
    localhost:3000/apis/provisioning.grafana.app/v0alpha1/namespaces/default/repositories/bootstrap \
    | jq '{deletionTimestamp: .metadata.deletionTimestamp, finalizers: .metadata.finalizers, health: .status.health}'
  ```

  If it stays past a few reconciles, the Grafana log names the resource the finalizer is failing on.
  Fixing the blocked contents is the right move; the rest of this is a forced removal and a last
  resort. Dropping the finalizers abandons the cleanup that had not finished, and it does **not**
  release what survives: those resources keep `grafana.app/managedBy`, `grafana.app/managerId` and
  their `grafana.app/sourcePath`, so they still point at a `bootstrap` repository that no longer
  exists, and the UI keeps treating them as managed. Releasing ownership is a separate, explicit
  step — strip those annotations — before anything is really unmanaged. Forced removal only
  applies to a repository already marked for deletion, which is the stuck case; on a live one the
  admission mutator puts the finalizers straight back:

  ```bash
  curl -su admin:admin -X PATCH \
    -H 'Content-Type: application/merge-patch+json' -d '{"metadata":{"finalizers":[]}}' \
    localhost:3000/apis/provisioning.grafana.app/v0alpha1/namespaces/default/repositories/bootstrap
  ```

  The usual cause is a managed folder holding resources the repository does not manage, which it
  will not delete. Move or delete those contents and the next reconcile finishes the folder; that
  fixes the cause rather than the symptom.

  `rm data/grafana.db` is a last resort for a disposable instance only. It discards the whole dev
  database — every dashboard, data source, user and org, not just this example — so back the
  file up first if any of it matters, and note that it does nothing for an instance configured
  to use Postgres, MySQL, or a database path other than the default.

- **This reroutes library panel storage for the whole instance.** `dualWriterMode = 5` reads from
  unified storage whatever the migration state, so library panels that exist only in the legacy
  `library_element` table are not listed while this config is in place, and panels created while it
  is in place are not listed once it is removed. Nothing is deleted in either direction — the rows
  stay put and reappear when the routing goes back. `./setup.sh provisioning-bootstrap` always asks
  before enabling it — it cannot count existing panels unless Grafana happens to be running, and
  the usual order is to set this up first — so answer `y`, or set `BOOTSTRAP_FORCE=1` when running
  it non-interactively. Prefer a scratch instance.
- Library panels and playlists are `:disabled` in the default `[provisioning] resources` set;
  `devenv/provisioning-bootstrap.ini` enables them, and serves library panels from unified
  storage so their manager annotations survive a round trip.
