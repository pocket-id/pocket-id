import playwrightConfig from "../playwright.config";

export async function cleanupBackend() {
  const response = await fetch(
    playwrightConfig.use!.baseURL + "/api/test/reset",
    {
      method: "POST",
    }
  );

  if (!response.ok) {
    throw new Error(
      `Failed to reset backend: ${response.status} ${response.statusText}`
    );
  }
}
