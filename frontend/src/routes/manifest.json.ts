import { getLogoUrl } from '$lib/utils/logo-util';

export async function GET({ request }) {
  const logoUrl = getLogoUrl();
  const manifest = {
    name: "PocketID",
    icons: [
      {
        src: logoUrl
      }
    ],
    display: "standalone",
    background_color: "#000000",
    theme_color: "#000000"
  };

  return new Response(JSON.stringify(manifest), {
    headers: {
      'Content-Type': 'application/json'
    }
  });
}
