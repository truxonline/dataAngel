#!/usr/bin/env python3
import os
from datetime import datetime

stories = [
    "1-1-configurer-data-guard-via-annotations-k8s",
    "1-2-init-container-detect-healthy-data",
    "1-3-restore-conditionnel-ou-skip",
    "1-4-cli-verify-backup-state",
    "2-1-sidecar-litestream-backup-sqlite",
    "2-2-sidecar-rclone-sync-filesystem",
    "2-3-graceful-shutdown-with-wal-flush",
    "3-1-pre-backup-validation-sqlite-yaml",
    "3-2-post-restore-validation",
    "4-1-s3-distributed-lock-implementation",
    "4-2-lock-ttl-steal-mechanism",
    "5-1-prometheus-metrics-exporter",
    "5-2-alerting-backup-failure",
    "5-3-alerting-restore-performed",
    "6-1-cli-verify-backup-state",
    "6-2-cli-force-release-lock",
]


def validate_story(story_id):
    story_path = f"_bmad-output/implementation-artifacts/{story_id}.md"
    try:
        with open(story_path, "r", encoding="utf-8") as f:
            story_content = f.read()
    except FileNotFoundError:
        return {
            "story_id": story_id,
            "status": "ERROR",
            "issues": ["Story file not found"],
        }

    story_num = story_id.split("-")[0]
    story_name = "-".join(story_id.split("-")[1:])

    checklist_patterns = [
        f"_bmad-output/test-artifacts/atdd-checklist-{story_id}.md",
        f"_bmad-output/test-artifacts/atdd-checklist-{story_num}.md",
        f"_bmad-output/test-artifacts/atdd-checklist-{story_num}-{story_name}.md",
        f"_bmad-output/test-artifacts/atdd-checklist-{story_num}-{story_id.split('-')[1]}.md",
    ]
    checklist_content = None
    for pattern in checklist_patterns:
        try:
            with open(pattern, "r", encoding="utf-8") as f:
                checklist_content = f.read()
                break
        except FileNotFoundError:
            continue

    if checklist_content is None:
        return {
            "story_id": story_id,
            "status": "ERROR",
            "issues": ["Checklist file not found"],
        }

    issues = []

    if "# Story" not in story_content:
        issues.append("Story title missing")

    if "## Acceptance Criteria" not in story_content:
        issues.append("Acceptance criteria section missing")

    if (
        "Given" not in story_content
        or "When" not in story_content
        or "Then" not in story_content
    ):
        issues.append("Acceptance criteria not in Gherkin format (Given/When/Then)")

    if "stepsCompleted" not in checklist_content:
        issues.append("Checklist missing stepsCompleted")

    if "step-05-validate-and-complete" not in checklist_content:
        issues.append("Checklist not completed")

    unit_test_path = f"pkg/{story_num}/{story_id}_test.go"
    integration_test_path = f"pkg/{story_num}/{story_id}_integration_test.go"

    if not os.path.exists(unit_test_path):
        issues.append(f"Unit test file missing: {unit_test_path}")

    if not os.path.exists(integration_test_path):
        issues.append(f"Integration test file missing: {integration_test_path}")

    if os.path.exists(unit_test_path):
        with open(unit_test_path, "r") as f:
            test_content = f.read()
            if "t.Skip" not in test_content:
                issues.append("Unit tests not marked with t.Skip (TDD red phase)")

    if os.path.exists(integration_test_path):
        with open(integration_test_path, "r") as f:
            test_content = f.read()
            if "t.Skip" not in test_content:
                issues.append(
                    "Integration tests not marked with t.Skip (TDD red phase)"
                )

    if len(issues) == 0:
        status = "VALID"
    elif any("ERROR" in issue for issue in issues):
        status = "ERROR"
    else:
        status = "NEEDS_IMPROVEMENT"

    return {"story_id": story_id, "status": status, "issues": issues}


def main():
    print("=" * 80)
    print("VALIDATION DES STORIES TDD")
    print("=" * 80)
    print()

    results = []
    for story_id in stories:
        result = validate_story(story_id)
        results.append(result)

        status_icon = "❓"
        if result["status"] == "VALID":
            status_icon = "✅"
        elif result["status"] == "NEEDS_IMPROVEMENT":
            status_icon = "⚠️"
        elif result["status"] == "ERROR":
            status_icon = "❌"

        print(f"{status_icon} {story_id}")

        if result["issues"]:
            for issue in result["issues"]:
                print(f"   - {issue}")
        print()

    print("=" * 80)
    print("RÉSUMÉ")
    print("=" * 80)

    valid_count = sum(1 for r in results if r["status"] == "VALID")
    needs_improvement_count = sum(
        1 for r in results if r["status"] == "NEEDS_IMPROVEMENT"
    )
    error_count = sum(1 for r in results if r["status"] == "ERROR")

    print(f"✅ Valides: {valid_count}/{len(results)}")
    print(f"⚠️  À améliorer: {needs_improvement_count}/{len(results)}")
    print(f"❌ Erreurs: {error_count}/{len(results)}")
    print()

    if error_count > 0:
        print(
            "❌ Certaines stories ont des erreurs et doivent être corrigées avant le développement."
        )
    elif needs_improvement_count > 0:
        print("⚠️  Certaines stories ont des points à améliorer.")
    else:
        print("✅ Toutes les stories sont valides et prêtes pour le développement TDD.")
        print()
        print("PROCHAINE ÉTAPE:")
        print("1. Exécuter: /bmad-bmm-dev-story pour commencer le développement")
        print("2. Choisir une story à implémenter")
        print("3. Suivre le cycle TDD: Rouge → Vert → Refactor")


if __name__ == "__main__":
    main()
