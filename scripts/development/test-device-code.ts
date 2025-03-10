import axios from "axios";

const BASE_URL = "https://id.example.com/api";
const CLIENT_ID = "client-id";

interface DeviceAuthResponse {
  device_code: string;
  user_code: string;
  verification_uri: string;
  verification_uri_complete: string;
  expires_in: number;
  interval: number;
}

async function testDeviceFlow() {
  try {
    // Step 1: Request device code
    console.log("Requesting device code...");
    const { data } = await axios.post<DeviceAuthResponse>(
      `${BASE_URL}/oidc/device/authorize`,
      {
        client_id: CLIENT_ID,
        scope: "openid profile email",
      },
      {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
      }
    );

    console.log("\nDevice code received:");
    console.log("User Code:", data.user_code);
    console.log("Verification URI:", data.verification_uri);
    console.log("\nPlease visit the verification URI and enter the user code.");
    console.log("Or visit:", data.verification_uri_complete);

    // Step 2: Poll for tokens
    console.log("\nPolling for tokens...");
    const pollInterval = (data.interval || 5) * 1000; // Default to 5 seconds if interval is missing

    const poll = setInterval(async () => {
      try {
        const params = new URLSearchParams();
        params.append(
          "grant_type",
          "urn:ietf:params:oauth:grant-type:device_code"
        );
        params.append("device_code", data.device_code);
        params.append("client_id", CLIENT_ID);

        const tokenResponse = await axios.post(
          `${BASE_URL}/oidc/token`,
          params,
          {
            headers: {
              "Content-Type": "application/x-www-form-urlencoded",
            },
          }
        );

        if (tokenResponse.data.access_token) {
          console.log("\nAuthorization successful!");
          console.log("Access Token:", tokenResponse.data.access_token);
          console.log("ID Token:", tokenResponse.data.id_token);
          clearInterval(poll);
          process.exit(0);
        }
      } catch (error: any) {
        if (error.response?.data?.error === "authorization_pending") {
          process.stdout.write(".");
        } else {
          console.error("\nError:", error.response?.data || error.message);
          clearInterval(poll);
          process.exit(1);
        }
      }
    }, pollInterval);
  } catch (error: any) {
    console.error("Error:", error.response?.data || error.message);
    process.exit(1);
  }
}

testDeviceFlow();
