"""Regenerate literal conformance vectors without using the Go implementation.

The selected byte strings are authored below, rather than selected by a parser.
Python's hashlib result is cross-checked against OpenSSL before writing a fixture.
Go tests consume only the resulting JSON files and never run this script.
"""

import hashlib
import json
import pathlib
import struct
import subprocess

ROOT = pathlib.Path(__file__).parent
FRONTMATTER = "---\ntype: WorkItem\nid: work\ntitle: Work\ntriage: ready-for-agent\nexecution: unstarted\n---\n"


def fingerprint(tag, parts):
    parts = [part.encode("utf-8") for part in parts]
    encoded = tag.encode("utf-8") + b"\x00" + struct.pack(">Q", len(parts))
    for part in parts:
        encoded += struct.pack(">Q", len(part)) + part
    digest = hashlib.sha256(encoded).hexdigest()
    checked = subprocess.run(
        ["openssl", "dgst", "-sha256"], input=encoded, capture_output=True, check=True
    ).stdout.decode().strip().split()[-1]
    assert digest == checked
    return digest


def vector(name, body, ticket_parts, criteria_parts):
    fixture = {
        "source": FRONTMATTER + body,
        "ticketParts": ticket_parts,
        "criteriaParts": criteria_parts,
        "ticketSHA256": fingerprint("ctx.ticket-body.v1", ticket_parts),
        "criteriaSHA256": fingerprint("ctx.acceptance-criteria.v1", criteria_parts),
    }
    (ROOT / (name + ".json")).write_text(
        json.dumps(fixture, indent=2, ensure_ascii=False) + "\n", encoding="utf-8"
    )


intro = "# Ticket\n\nIntro.\n\n## Scope\nDeliver café.\n\n"
criteria = "## Acceptance criteria\n- [ ] Works.\n\n"
tail = "## Context\nNone\n"
vector(
    "basic",
    intro + criteria + "## Comments\nRoutine note.\n\n## Acceptance\n[Current](acceptance.md)\n\n" + tail,
    [intro + criteria + tail],
    [criteria],
)

intro = "# Ticket\n\n[Second][b] then [First][a].\n\n"
criteria = "## Acceptance criteria\n- [First][a] and [Second][b], again [First][a].\n\n"
definition_b = '[b]: second.md "B"\n'
definition_a = "[a]: first.md\n"
vector(
    "external-references",
    intro + criteria + "## Comments\n" + definition_b + definition_a + "[a]: ignored.md\nUnrelated note.\n",
    [intro + criteria, definition_b, definition_a],
    [criteria, definition_a, definition_b],
)

intro = "# Préface\r\n\r\n~~~md\r\n## Comments\r\nKeep fenced content.\r\n~~~\r\n\r\n"
criteria_1 = "## Acceptance criteria\r\n- [x] First.\r\n### Comments\r\nNested café.\r\n\r\n"
scope = "## Scope\r\nBody remains.\r\n\r\n"
criteria_2 = "Acceptance criteria\r\n-------------------\r\n- [ ] Second.\r\n\r\n"
appendix = "# Appendix\r\nRetained after the excluded section.\r\n"
vector(
    "section-boundaries-crlf",
    intro + criteria_1 + "## Comments\r\nHidden.\r\n### Nested\r\nStill hidden.\r\n" + scope
    + "## Comments\r\nRepeated exclusion.\r\n\r\n" + criteria_2 + "## Acceptance\r\nIgnored.\r\n"
    + "## Acceptance\r\nRepeated current link.\r\n" + appendix,
    [intro + criteria_1 + scope + criteria_2 + appendix],
    [criteria_1, criteria_2],
)

inside = "[inside]: inside.md\n"
scope = "## Scope\n[Visible][inside].\n\n" + inside + "\n"
criteria = "## Acceptance criteria\n- [Visible][inside], ![Diagram][diagram], [shortcut], and [collapsed][].\n\n"
diagram = '[diagram]: <diagram.svg>\n  "Two lines\n  of title"\n'
shortcut = "[shortcut]: shortcut.md\n"
collapsed = "[collapsed]: collapsed.md\n"
vector(
    "retained-and-image-references",
    scope + criteria + "## Comments\n" + diagram + shortcut + collapsed + "[inSIDE]: ineffective.md\n",
    [scope + criteria, diagram, shortcut, collapsed],
    [criteria, inside, diagram, shortcut, collapsed],
)

criteria = "## Acceptance criteria\n- [Rule][rule].\n"
effective = '[rule]: rule.md\n  "Title\n  "\n'
duplicate = "## Appendix\n[rule]: ignored.md\n"
vector(
    "earlier-excluded-definition",
    "## Comments\n" + effective + criteria + duplicate,
    [criteria + duplicate, effective],
    [criteria, effective],
)

criteria = "## Acceptance criteria\r\n- [empty][] and [empty title][title].\r\n\r\n"
empty = "[empty]: <>\r\n"
title = '[title]: destination.md\r\n  ""'
vector(
    "crlf-references-without-final-newline",
    criteria + "## Comments\r\n" + empty + title,
    [criteria, empty, title],
    [criteria, empty, title],
)
