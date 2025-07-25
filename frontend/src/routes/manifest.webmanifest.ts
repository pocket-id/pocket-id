import type { RequestHandler } from './$types';
import { getAppleIconUrl } from '$lib/utils/apple-icon-util';
import type { RequestEvent } from '@sveltejs/kit';

export async function GET({ request }: RequestEvent) {
  const logoUrl = getAppleIconUrl();
  const manifest = {
    name: "PocketID",
    icons: [
      {
        src: logoUrl
      }
    ],
    display: "browser",
    background_color: "#000000",
    theme_color: "#000000"
  };

  return new Response(JSON.stringify(manifest), {
    headers: {
      'Content-Type': 'application/manifest+json'
    }
  });
}
