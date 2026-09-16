# Discovery configuration verification

Observed by workflow actor `Codex /root` on `2026-09-17T00:25:08+02:00`.

Tested source identity: `working-tree:a0d474ff98dc483343da5c1585405e3588583b9f; source-manifest-sha256:189ab90d39375fa772d807b6a3388461dd4b63e71fada39ec2663b3f86d1eb19`.

The Go source and test-fixture manifest below identifies the working tree used for the checks. It includes the new untracked configuration package. The manifest was captured after checks, with no intervening code changes. README and workflow records were subsequently updated for completion. This is local verification and a workflow acceptance decision; it asserts no human review or approval.

## Criteria results

1. **Pass.** `TestReadSharedConfiguration`, `TestReadPersonalConfiguration`, and `TestSharedDeclarationsWorkIndependently` parse both profiles, project bindings, aliases, workspace declarations, and ordered members. `TestConfigurationFieldErrors` checks the declaring file and exact nested field on malformed declarations.

2. **Pass.** `TestConfigurationFieldErrors` and `TestFrontmatterAndVersionErrors` reject malformed and duplicate-key YAML, known-field type errors, missing required identities, mixed profiles, unsupported versions, duplicate registration keys, duplicate aliases within each selector kind, and duplicate member keys. `TestReadSharedConfiguration` and `TestUnknownYAMLSyntaxRemainsInOriginalSource` retain CRLF Markdown, unknown nested metadata, comments, tags, anchors, and original scalar spelling through captured source text.

3. **Pass.** `TestConfigSymlinkUsesEncounteredDirectory` retains the declaring-file base even when both the encountered and target directories have usable records. `TestTargetAliasesAndLiteralPaths` resolves target aliases and treats tilde, environment-looking values, commas, and percent escapes literally. `TestTargetFailuresRemainAttributable` preserves missing-target, symlink-loop, and non-directory traversal failures on their fields while retaining a usable unrelated project. Canonical identity remains separate from selected-directory validation.

4. **Pass.** The external-package tests use real temporary filesystems, independent record directories without Git, malformed input, both profiles, duplicate declarations, relative and absolute paths, and symlinks. `TestReadSharedConfiguration` compares the original file and directory entries and verifies that missing targets were not created.

## Pickup and scope

The current reader was built before pickup and received every full source returned by the task-context command, together with its path and inclusion reasons. The command exited 0 with both completeness fields true and no diagnostics. After updating Current commitments, orientation exited 0, was complete, had no diagnostics, and selected ticket 01 as its sole eligible item. Its criteria, selected context, dependencies, and blocking decisions all passed.

```sh
go build -o .cache/discovery-01/ctx ./cmd/ctx
.cache/discovery-01/ctx context --project .scratch/records --ticket project-discovery/issues/01-read-configuration.md --allow-source .
.cache/discovery-01/ctx orient --project .scratch/records --allow-source . --json
```

The actual context invocation used absolute project, binary, and allowed-source paths rooted at `/Users/zoki/code/context`. Initial selected source digests:

| Source | SHA-256 |
| --- | --- |
| `.scratch/records/project-discovery/issues/01-read-configuration.md` | `77c7354a7697fbe4d854ae960adc74f7dad3a08ab70651992a839b152067ebaa` |
| `.scratch/project-discovery/spec.md` | `d54c182412a9ee711e1b7a91e2a5ea6c4174dbd8567304ff44fa109a65c41d67` |
| `docs/vision.md` | `d8d7fb88699c3310cd5f489d6d44dbf342fe813c372c54485400f6765f5aa2e9` |
| `CONTEXT.md` | `a003af0f4845f28fe14191dec210c7ceeabf6a5422fb51e7e0ee7e957d366a55` |
| `docs/adr/0004-directory-specific-project-selection.md` | `efd4654de6c787345c341c26a0076fb605214b3d9aeecc3a1ec01f93752cefbf` |
| `docs/agents/acceptance.md` | `fbde78ab62733d1bbe091fba6a867f30f50076c819ee3b0783ac4dcd2d7bf654` |

## Commands and results

```text
go test ./...
exit: 0
?   	github.com/Zokiio/context/cmd/ctx	[no test files]
ok  	github.com/Zokiio/context/internal/cli	(cached)
ok  	github.com/Zokiio/context/internal/discoveryconfig	0.278s
ok  	github.com/Zokiio/context/internal/orientation	(cached)
?   	github.com/Zokiio/context/internal/recordread	[no test files]
ok  	github.com/Zokiio/context/internal/taskcontext	(cached)
```

```text
go test -race ./...
exit: 0
?   	github.com/Zokiio/context/cmd/ctx	[no test files]
ok  	github.com/Zokiio/context/internal/cli	5.570s
ok  	github.com/Zokiio/context/internal/discoveryconfig	1.257s
ok  	github.com/Zokiio/context/internal/orientation	2.026s
?   	github.com/Zokiio/context/internal/recordread	[no test files]
ok  	github.com/Zokiio/context/internal/taskcontext	(cached)
```

```text
go vet ./...
exit: 0
```

```text
go build -o .cache/discovery-01/ctx-verified ./cmd/ctx
exit: 0
```

No selector, ancestor-discovery, workspace-navigation, or setup behavior changed in this slice. Those behaviors have their own tickets. The CLI build verifies the existing entry point; the new package is exercised directly by its tests.

## Source manifest

The digest above is SHA-256 over UTF-8 JSON of this mapping with keys sorted and separators `,` and `:`, without spaces or a trailing newline. Each value is the whole-file SHA-256. This includes source code and the fixtures used by repository tests.

```json
{
  "cmd/ctx/main.go": "1abd2a3fb3b3374c07b72da7f004a2ff658e82dcd09847777d9079d56e8fdcdf",
  "go.mod": "e5647064770acf766548cdf4ed478568e99d933d3e02741f0a63ac062db65430",
  "go.sum": "c807352ac48af4ebb9e3dfc5619472566f10d5814d85b04508d006327670cf1e",
  "internal/cli/acceptance.go": "949d5912e870712227931aa3081fd4b66b535b3c2e2761e0d16066612c416009",
  "internal/cli/acceptance_cli_test.go": "f82f525ac3b0dec1f73513fe6b44a38f2569f954d4597cf825c0f1908c936beb",
  "internal/cli/cli.go": "eb268e11afc7aa483993277a7b7cd3b519b48e28eb5be6938f521d013b7dce81",
  "internal/cli/cli_test.go": "345d781248c34176aa5f0f812d58908a10e6f39ca07d26c2813a7a37ccaf7016",
  "internal/cli/dependencies_cli_test.go": "c3705b6409393eecacd275d01934cf314a441b2bca2fe2874f5317c0a794e710",
  "internal/cli/orientation.go": "758ec5ee3756687083012810dffd9481aa4ddef95d536d96859bddf94c8f9f8e",
  "internal/cli/orientation_cli_test.go": "84aa6576f2c7248d01db3123bf69dcff10f6ba64e0ba50745a27e905b2e68df9",
  "internal/cli/testdata/blockers.txt": "d992d406f1bc1becf6a12f0d9bc2a91675b6b29201974597d73ffe88d459beb3",
  "internal/cli/testdata/context.txt": "12daccc57f1d02d513fbf201cad89ba2e30e929c56a953f65ce1f36dba5bde58",
  "internal/cli/testdata/external.txt": "78cbf763ac7a1d30c2c422a3879d8aedc5fceddab195cdf7272608716f30fb46",
  "internal/cli/testdata/limits.txt": "03e8dbd392d041885bf09e4b8bdef57644b4453ea617cf93d8a4e1d8cfd1337a",
  "internal/cli/testdata/linked.txt": "e5ef0618ae25860ed23cbacd6644b8d65f1a31e780b80e9b9691716fea4d943a",
  "internal/cli/testdata/orient-acceptance.txt": "fb5263c5dd75823a8cf40018811313696273c1b46a598f8a1f76f2b01012369e",
  "internal/cli/testdata/orient-cycles.txt": "2017af8e79a37169b17fd98a43b943f0dc10d3a65b94b9921eb636a3260077fd",
  "internal/cli/testdata/orient-decisions.txt": "1af679e0bf7f2edc65a34456acb4d5a2bfa112489590c59e0b7a6086aef53d59",
  "internal/cli/testdata/orient-failures.txt": "c2f8490a88d6cbbd760f429243be758d486db765e55da81dc34b897e4c6fc110",
  "internal/cli/testdata/orient-fingerprints.txt": "b5745edd682f96a1716e9afe12ca5471501826c1420d5cf10d6dd932f041ddd2",
  "internal/cli/testdata/orient-inventory.txt": "ccf6d7bc835128f6c698cb0fd9d6ad81e2f79680836c480bc36efefc379efdfb",
  "internal/cli/testdata/orient-readiness.txt": "82f84ad91038c16d5e2ea61068294fbb9d80163d28b15e333c33284c790eec6e",
  "internal/cli/testdata/orient.txt": "bcd162becc4c6bc15b6e9e0048dc84d506d45dbfe6bb175379f3ce22c0908583",
  "internal/discoveryconfig/config.go": "4df7f2847013f0b92386a764320916eca8b9043af3297644a7cd5bc83ee91db1",
  "internal/discoveryconfig/config_test.go": "d49f8a166a1bb7f7b5c6e256966e9c91df15172f7a2d521e3459901025625999",
  "internal/discoveryconfig/paths.go": "d81b790b150b9fb754c4c781e8f623e9497c5663676ebe47fc3636a427af98b1",
  "internal/discoveryconfig/paths_test.go": "062d395eb5e8338c08f8ecc93d28583d97c3f27e7b1c8d58087bc5e824e6bd2c",
  "internal/discoveryconfig/profiles.go": "95ab75d99faa7a0c44b0abc5e045ba053c28ef83855d3cc319efdfd5e4f10d99",
  "internal/orientation/acceptance.go": "10f411ec16a73d70c90e5ff41b2a023b3268a5181d7a48a187e1e94f8f6ff375",
  "internal/orientation/acceptance_snapshots.go": "474a44d7d3863cfa8318c0664ab3a69c7efa42b78755e793971f29ef01c5a847",
  "internal/orientation/acceptance_test.go": "1929a547cc305a10fdc591173b2d6ba3c3502936cf0ddf093ce69455c8a2f09d",
  "internal/orientation/checks.go": "f67876785e3cbfc9e347db628c571531039bac6aea56916acd31bf1208e98688",
  "internal/orientation/decisions.go": "cb2ee48e4f4a1c3063e7242f34a871256c7c2f84081c730507b5e3ecac564f98",
  "internal/orientation/decisions_test.go": "985d3d042ea24300453e17d97905c5b22d5b5b5675f0d406b3f2fc29c03952b6",
  "internal/orientation/dependencies.go": "b0b13c3527bd16dea1eb4d35a450c91d0d655d7234855f731fc5c277924c3aa6",
  "internal/orientation/dependencies_test.go": "92266e89c891bc7c2c2951240cad93236651d7aa106984cb2c266e3e11a0b3dc",
  "internal/orientation/fingerprint.go": "8956e2d48b7023df852e152d400abd4a5847a98b13164cc38f912d5669c986ed",
  "internal/orientation/fingerprint_test.go": "93bde08233ef7c9afb79502241702c90346168cc782edad067de6322933b1c37",
  "internal/orientation/inventory.go": "62c02eaaf1e759949107388441ec8ef13188a007ced4af560acf4b71526f94b0",
  "internal/orientation/limits_test.go": "60edcd80cd085bc8c6f4f27db90b9b4b28346fc96d665c3a2606bff275f2f2f6",
  "internal/orientation/metadata.go": "1e07a3d2a30ee25603db998e606396cf0913e683e456e976af0c90aaa468745c",
  "internal/orientation/orientation.go": "011cbc2e943225fd8fdd3925e0a0baa6e568edcadb0a376782bc793e7b331901",
  "internal/orientation/orientation_test.go": "a04a11b9764f09f1c45fbd3c4aa5c0105ca50e72dbd42d89c7200c041d528753",
  "internal/orientation/profiles.go": "938260d63d3d6c8175a93f4a9a4a5d8c6650f730edd7bfe6546f3ae662797fde",
  "internal/orientation/readiness.go": "e27f7b4bece7271015f59e1fd95c7ae7077f0d198e32f1e869dace92256870c9",
  "internal/orientation/readiness_test.go": "b3fe7e7b98f6dad2dab314d1f622cd00d61d5aae8d2b43879208716643c59e86",
  "internal/orientation/sources_test.go": "e7a28851839b78a8337ed390ba3af0a96f21bf753b19a7c7af5bbe50b56c2a8e",
  "internal/orientation/testdata/fingerprints/basic.json": "93b8e63e2078d537d4d830bb108260e18e4bfce86db8d8626c3890fcd8919179",
  "internal/orientation/testdata/fingerprints/crlf-references-without-final-newline.json": "3641bdc7ba3919a123b96dcaf3fcc30e1c97765ce4f60c75bc429c281bf5e6f9",
  "internal/orientation/testdata/fingerprints/earlier-excluded-definition.json": "720384155daa64f87c52136104e7507181104e4f1bfb16e3eacf5d8c0b955f91",
  "internal/orientation/testdata/fingerprints/external-references.json": "73753c7c45f6767779004746e0675f95baa4d826bd7b9eca06b9fe750d26f271",
  "internal/orientation/testdata/fingerprints/generate.py": "3954c23cf4afb3e1f2774e15903575cf1558bed1eb3f872044573693059ec455",
  "internal/orientation/testdata/fingerprints/retained-and-image-references.json": "637dc355bea6ec36b8f7a694f2fe9a676a3b15a10f78ccbb6ff40e785380b44d",
  "internal/orientation/testdata/fingerprints/section-boundaries-crlf.json": "c0d5848ab516e8235bf4feebbb68ffdf3358d9f8649ea6f440a857f469047cdb",
  "internal/orientation/types.go": "2c9642c3ef98c1e79146ac3e3995dfbe9201019d52bb2b7cf0e681023f6efe3b",
  "internal/recordread/document.go": "34e45d4754c74b685c2cc4f8699bc9043525717e1523eeb5d0095131b26418f4",
  "internal/recordread/sections.go": "ea7980e8ba2252d9ea8d6e3dded8b637ce0df23e514271a4f5804a1e5becc621",
  "internal/recordread/source.go": "9d744bd6ad6e70ab678fcf85beb691bf551cd0f94c15c1884371fee1182e0ab8",
  "internal/recordread/types.go": "a0b4b97469fa25cf662e172202cfca9ecf6b55fb207c4ecb96610154466e7682",
  "internal/taskcontext/context.go": "941fe6224fa61e262e6715476db1d802a100cec978c1988c87d5dcca35a252e7",
  "internal/taskcontext/context_test.go": "cf6c74b0ec0f07ab6dff66591c4671dce837512edc9471a98bc1d99d721f78e4",
  "internal/taskcontext/document.go": "ab086df322ce66472ac579ffebb5f35e46bc723b9605bb8ce79b80182fba0f8e",
  "internal/taskcontext/external_test.go": "c584db2bb525722b2544e1591b8c1533cf370fc395cd1c146fb4cf5296524571",
  "internal/taskcontext/limits_test.go": "adae438e4be3c1c08b74c00d6f525dd94de5c3cd1573ec5b2c15d6431a9167e5",
  "internal/taskcontext/options.go": "2ec4a57ab69d124e969beff34df154a4f98e4217df6257ba72525b66531bf0e2",
  "internal/taskcontext/relationships.go": "697166dc52dbb47e575fcfd85e29965ec4149162b85f8c93951457a048879b3b",
  "internal/taskcontext/relationships_test.go": "4cd28de60010f7db4080b974efc2c3aa3832ce902508e3a6c4ed0541644a363a",
  "internal/taskcontext/source.go": "809bf450400cdc2c2d02975302fa556f56af5f63a4e1aa92d88e4695577b474f",
  "internal/taskcontext/traversal_test.go": "f4bf4e26f0f7560510a88ac362681cd81c00853e84963355addf66e26a0a3580"
}
```
