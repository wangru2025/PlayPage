import { AuthGate } from "./AuthGate";

export default function AuthPage() {
  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <AuthGate />
    </main>
  );
}
