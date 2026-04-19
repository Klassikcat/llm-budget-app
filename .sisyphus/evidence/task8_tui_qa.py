import os
import re
import sys
import time
from pathlib import Path

import pexpect


ROOT = Path(__file__).resolve().parents[2]
EVIDENCE_DIR = ROOT / ".sisyphus" / "evidence"
RAW_PATH = EVIDENCE_DIR / "task8-tui-graph-flow.raw.txt"
SUMMARY_PATH = EVIDENCE_DIR / "task8-tui-graph-flow.txt"
SNAPSHOT_PATH = EVIDENCE_DIR / "task8-tui-graph-flow.snapshots.txt"

ANSI_RE = re.compile(r"\x1B(?:\[[0-?]*[ -/]*[@-~]|[@-Z\\-_])")


def strip_ansi(text: str) -> str:
    return ANSI_RE.sub("", text)


def append_step(lines: list[str], message: str) -> None:
    lines.append(f"- {message}")


def try_expect(child: pexpect.spawn, pattern: str, timeout: int) -> bool:
    try:
        child.expect(pattern, timeout=timeout)
        return True
    except Exception:
        return False


def capture_context(text: str, marker: str, radius: int = 500) -> str:
    idx = text.find(marker)
    if idx == -1:
        return "(marker not found)"

    start = max(0, idx - radius)
    end = min(len(text), idx + len(marker) + radius)
    snippet = text[start:end].strip()
    return snippet or "(empty snippet)"


def main() -> int:
    steps: list[str] = []
    matched_markers: dict[str, str] = {
        "model_token_usage": "",
        "model_cost": "",
        "daily_token_trend": "",
        "token_breakdown": "",
    }
    observed: dict[str, bool] = {
        "entered_graph_mode": False,
        "saw_model_token_usage": False,
        "saw_model_cost": False,
        "saw_daily_token_trend": False,
        "saw_token_breakdown": False,
        "returned_to_dashboard": False,
        "quit_cleanly": False,
    }

    RAW_PATH.parent.mkdir(parents=True, exist_ok=True)

    with RAW_PATH.open("w", encoding="utf-8") as raw_file:
        child = pexpect.spawn(
            "go",
            ["run", "./cmd/tui"],
            cwd=str(ROOT),
            encoding="utf-8",
            codec_errors="replace",
            timeout=10,
            dimensions=(40, 140),
        )
        child.logfile_read = raw_file

        try:
            append_step(steps, "Launched `go run ./cmd/tui` under pexpect with a pseudo-TTY.")
            dashboard_ready = try_expect(child, "Recent Sessions", timeout=15) or try_expect(child, "m manual form", timeout=5)
            append_step(steps, f"Waited for dashboard readiness marker: {dashboard_ready}.")

            child.send("g")
            append_step(steps, "Sent `g` to enter graph mode.")
            graph_ready = try_expect(child, "\\[Model Token Usage\\]", timeout=10) or try_expect(child, "Model Token Usage", timeout=10)
            if graph_ready:
                matched_markers["model_token_usage"] = "[Model Token Usage]"
            append_step(steps, f"Observed graph-mode marker after `g`: {graph_ready}.")

            child.send("\t")
            append_step(steps, "Sent first `Tab` to move to the next graph tab.")
            first_tab_seen = try_expect(child, "\\[Model Cost\\]", timeout=10) or try_expect(child, "Model Cost", timeout=10)
            if first_tab_seen:
                matched_markers["model_cost"] = "[Model Cost]"
            append_step(steps, f"Observed first tab transition marker: {first_tab_seen}.")

            child.send("\t")
            append_step(steps, "Sent second `Tab` to move to the next graph tab.")
            second_tab_seen = try_expect(child, "\\[Daily Token Trend\\]", timeout=10) or try_expect(child, "Daily Token Trend", timeout=10)
            if second_tab_seen:
                matched_markers["daily_token_trend"] = "[Daily Token Trend]"
            append_step(steps, f"Observed second tab transition marker: {second_tab_seen}.")

            child.send("\t")
            append_step(steps, "Sent third `Tab` to move to the Token Breakdown tab and waited for a breakdown-specific content marker.")
            third_tab_seen = try_expect(child, "C\\.Read:", timeout=10) or try_expect(child, "C\\.Write:", timeout=10) or try_expect(child, "No token breakdown data for this month\\.", timeout=10)
            if third_tab_seen:
                matched_markers["token_breakdown"] = "C.Read:"
            append_step(steps, f"Observed third tab transition marker: {third_tab_seen}.")

            time.sleep(1)
            append_step(steps, "Allowed the Token Breakdown tab to remain visible briefly before exiting graph mode.")

            child.send("\x1b")
            append_step(steps, "Sent `Esc` to return to the dashboard.")
            dashboard_returned = try_expect(child, "m manual form", timeout=5) or try_expect(child, "Recent Sessions", timeout=5)
            append_step(steps, f"Observed dashboard marker after `Esc`: {dashboard_returned}.")

            child.send("q")
            append_step(steps, "Sent `q` to quit the TUI.")
            try:
                child.expect(pexpect.EOF, timeout=5)
                observed["quit_cleanly"] = True
            except Exception:
                child.send("\x03")
                append_step(steps, "Sent `Ctrl+C` fallback after `q` did not terminate the session in time.")
                child.expect(pexpect.EOF, timeout=5)
        except Exception as exc:
            append_step(steps, f"Automation hit an exception: {exc!r}")
        finally:
            if child.isalive():
                child.close(force=True)

    raw_text = RAW_PATH.read_text(encoding="utf-8")
    clean_text = strip_ansi(raw_text)

    observed["entered_graph_mode"] = "Active tab:" in clean_text or "Model Token Usage" in clean_text
    observed["saw_model_token_usage"] = "Active tab: Model Token Usage" in clean_text or "[Model Token Usage]" in clean_text
    observed["saw_model_cost"] = "Active tab: Model Cost" in clean_text or "[Model Cost]" in clean_text
    observed["saw_daily_token_trend"] = "Active tab: Daily Token Trend" in clean_text or "[Daily Token Trend]" in clean_text
    observed["saw_token_breakdown"] = "[Token Breakdown]" in clean_text and (
        "C.Read:" in clean_text or "C.Write:" in clean_text or "No token breakdown data for this month." in clean_text
    )
    observed["returned_to_dashboard"] = "m manual form" in clean_text or "Dashboard" in clean_text

    tab_snapshots = {
        "Model Token Usage": capture_context(clean_text, "[Model Token Usage]"),
        "Model Cost": capture_context(clean_text, "[Model Cost]"),
        "Daily Token Trend": capture_context(clean_text, "[Daily Token Trend]"),
        "Token Breakdown": capture_context(clean_text, "C.Read:") if "C.Read:" in clean_text else capture_context(clean_text, "No token breakdown data for this month."),
    }

    limitations: list[str] = []
    if "\x1b[?1049h" in raw_text or "\x1b[?1049l" in raw_text:
        limitations.append("Bubble Tea used the terminal alternate screen buffer, so the raw capture contains control sequences and partial redraws.")
    if not observed["entered_graph_mode"]:
        limitations.append("The capture did not expose a stable graph-mode marker even though key events were sent through a pseudo-TTY.")
    missing_tabs = [
        label
        for label, key in [
            ("Model Token Usage", "saw_model_token_usage"),
            ("Model Cost", "saw_model_cost"),
            ("Daily Token Trend", "saw_daily_token_trend"),
            ("Token Breakdown", "saw_token_breakdown"),
        ]
        if not observed[key]
    ]
    if missing_tabs:
        limitations.append("Not every graph tab produced a clearly matchable text marker in the captured stream: " + ", ".join(missing_tabs) + ".")

    summary_lines = [
        "Task 8 TUI graph-flow QA summary",
        "",
        "Command:",
        "- go run ./cmd/tui",
        "",
        "Automated steps:",
        *steps,
        "",
        "Observed markers:",
    ]
    for key, value in observed.items():
        summary_lines.append(f"- {key}: {value}")

    summary_lines.extend([
        "",
        "Per-tab evidence snippets:",
    ])
    for label, snippet in tab_snapshots.items():
        summary_lines.extend([
            f"- {label}:",
            f"  {snippet.replace(chr(10), chr(10) + '  ')}",
        ])

    summary_lines.extend([
        "",
        "Limitations:",
    ])
    if limitations:
        summary_lines.extend(f"- {item}" for item in limitations)
    else:
        summary_lines.append("- None observed in this environment.")

    summary_lines.extend([
        "",
        "Raw capture:",
        f"- {RAW_PATH.relative_to(ROOT)}",
        "Focused snapshots:",
        f"- {SNAPSHOT_PATH.relative_to(ROOT)}",
    ])

    snapshot_lines = ["Task 8 TUI graph-flow focused snapshots", ""]
    for label, snippet in tab_snapshots.items():
        snapshot_lines.extend([
            label,
            "-" * len(label),
            snippet,
            "",
        ])

    SNAPSHOT_PATH.write_text("\n".join(snapshot_lines).rstrip() + "\n", encoding="utf-8")
    SUMMARY_PATH.write_text("\n".join(summary_lines) + "\n", encoding="utf-8")
    sys.stdout.write(str(SUMMARY_PATH.relative_to(ROOT)) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
