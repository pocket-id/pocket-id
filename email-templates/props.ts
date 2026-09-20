export interface SharedProps {
  logoURL: string;
  appName: string;
  appURL: string;
}

export const sharedPreviewProps: SharedProps = {
  logoURL: "https://pocket-id.org/img/logo.png",
  appName: "Pocket ID",
  appURL: "https://id.example.com",
};

export const sharedTemplateProps: SharedProps = {
  logoURL: "{{.LogoURL}}",
  appName: "{{.AppName}}",
  appURL: "{{.AppURL}}",
};
