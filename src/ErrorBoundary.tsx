import type { ErrorInfo, ReactNode } from "react";
import { Component } from "react";
import { clearStoredSessionState } from "./storageKeys";

interface Props {
  children: ReactNode;
}

interface State {
  failed: boolean;
  message: string;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { failed: false, message: "" };

  static getDerivedStateFromError(error: Error) {
    return { failed: true, message: error.message };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("App render failed", error, info);
  }

  render() {
    if (!this.state.failed) return this.props.children;

    return (
      <main className="app-shell fallback-shell">
        <section className="fallback-panel">
          <strong>화면을 복구할 수 없습니다.</strong>
          <span>저장된 로컬 방/세션 정보를 비우고 다시 시작해 주세요.</span>
          {this.state.message ? <code>{this.state.message}</code> : null}
          <button
            onClick={() => {
              clearStoredSessionState();
              window.location.reload();
            }}
          >
            로컬 세션 초기화
          </button>
        </section>
      </main>
    );
  }
}
