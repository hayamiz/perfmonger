# Archive

`TODO.md` と `DONE.md` は Go 移行プロジェクトの進捗管理に使われていた
自由形式のタスクリスト（旧 `devdoc/TODO.md` / `devdoc/DONE.md` 由来）。

2026-04-20 時点で `ticket` プラグイン運用に移行したため、歴史的経緯と
完了タスクの記録としてここに保管する。現行のオープンなタスクは
`doc/tickets/NNNN-*.md` に分解済みで、以下の対応関係:

- 「pager 機能の有効化」 → [../0001-enable-pager-in-summary.md](../0001-enable-pager-in-summary.md)
- 「player の RunDirect API 実装」 → [../0002-player-rundirect-api.md](../0002-player-rundirect-api.md)
- 「summarizer の RunDirect API 実装」 → [../0003-summarizer-rundirect-api.md](../0003-summarizer-rundirect-api.md)
- 「Ruby コードの撤去」 → [../0004-remove-ruby-code.md](../0004-remove-ruby-code.md)
- 「README.md の更新」 → [../0005-update-readme.md](../0005-update-readme.md)
- 「Rakefile の置換」 → [../0006-replace-rakefile.md](../0006-replace-rakefile.md)
- 「クロスビルド・リリース生成」 → [../0007-release-pipeline.md](../0007-release-pipeline.md)

新しいタスクは `/ticket-create` で追加すること。これらのファイルは
今後更新しない。
