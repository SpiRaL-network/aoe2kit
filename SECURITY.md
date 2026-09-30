# Security

AoE2Kit is an offline file-analysis and authoring toolkit. It should not need
credentials to inspect ordinary scenarios, replays, data files, or local mods.

## Reporting

For public releases, report security issues through the project's GitHub issue
or security-advisory channel. Include the command, input file type, and a small
reproducer when possible. Do not attach private replays, private mods, or files
containing personal information unless you are comfortable sharing them with the
maintainer.

## File Safety

- Treat third-party `.aoe2record`, `.aoe2scenario`, `.dat`, `.sld`, `.xs`, `.per`,
  and mod folders as untrusted input.
- Prefer read-only commands before write commands.
- Use `kit scen write-check`, `kit dat roundtrip`, and relevant `delete-plan`
  commands before destructive edits.
- Keep original game files backed up outside the working folder.

## Privacy

Replays and scenarios can contain player names, chat, authored text, mod names,
and other identifying information. AoE2Kit tries to make those facts visible,
not hidden. Review outputs before publishing reports or fixtures.
