import playwrightConfig from "../playwright.config";

export async function cleanupBackend() {
  const url = new URL("/api/test/reset", playwrightConfig.use!.baseURL);

  if (process.env.SKIP_LDAP_TESTS === "true") {
    url.searchParams.append("skip-ldap", "true");
  }

  const response = await fetch(url, {
    method: "POST",
  });

  if (!response.ok) {
    throw new Error(
      `Failed to reset backend: ${response.status} ${response.statusText}`
    );
  }
}
