import { render } from '@react-email/components';
import { NewSignInEmail } from './emails/new-signin-email';
import { OneTimeAccessEmail } from './emails/one-time-access-email';
import { ApiKeyExpiringEmail } from './emails/api-key-expiring-email';
import { TestEmail } from './emails/test-email';
import * as fs from 'node:fs';
import * as path from 'node:path';

const outputDir = '../';

if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

function cleanupOldTemplates() {
  console.log('Cleaning up old template files...');
  
  try {
    const files = fs.readdirSync(outputDir);
    const templateFiles = files.filter(file => file.endsWith('.tmpl'));
    
    for (const file of templateFiles) {
      const filePath = path.join(outputDir, file);
      fs.unlinkSync(filePath);
      console.log(`✓ Removed ${file}`);
    }
    
    if (templateFiles.length === 0) {
      console.log('No old template files to clean up');
    } else {
      console.log(`Cleaned up ${templateFiles.length} old template files`);
    }
  } catch (error) {
    console.error('Error cleaning up old templates:', error);
  }
}

const templates = [
  {
    name: 'login-with-new-device',
    component: NewSignInEmail,
    sampleProps: {
      logoURL: "LOGO_URL_PLACEHOLDER",
      appName: "APP_NAME_PLACEHOLDER",
      data: {
        city: "CITY_PLACEHOLDER",
        country: "COUNTRY_PLACEHOLDER",
        ipAddress: "IP_ADDRESS_PLACEHOLDER",
        device: "DEVICE_PLACEHOLDER",
        dateTime: "DATE_TIME_PLACEHOLDER",
      }
    },
    replacements: [
      { search: /LOGO_URL_PLACEHOLDER/g, replace: '{{.LogoURL}}' },
      { search: /APP_NAME_PLACEHOLDER/g, replace: '{{.AppName}}' },
      { search: /CITY_PLACEHOLDER/g, replace: '{{.Data.City}}' },
      { search: /COUNTRY_PLACEHOLDER/g, replace: '{{.Data.Country}}' },
      { search: /IP_ADDRESS_PLACEHOLDER/g, replace: '{{.Data.IPAddress}}' },
      { search: /DEVICE_PLACEHOLDER/g, replace: '{{.Data.Device}}' },
      { search: /DATE_TIME_PLACEHOLDER/g, replace: '{{.Data.DateTime.Format "January 2, 2006 at 3:04 PM MST"}}' },
    ],
    conditionals: [
      {
        regex: /(<td[^>]*>[\s\S]*?<.*?>Approximate Location<[\s\S]*?{{\.Data\.City}}, {{\.Data\.Country}}[\s\S]*?<\/td>)/,
        wrapper: '{{if and .Data.City .Data.Country}}$1{{end}}'
      }
    ],
    textTemplate: `{{ define "root" -}}
New Sign-In Detected
====================

{{ if and .Data.City .Data.Country }}
Approximate Location: {{ .Data.City }}, {{ .Data.Country }}
{{ end }}
IP Address: {{ .Data.IPAddress }}
Device:     {{ .Data.Device }}
Time:       {{ .Data.DateTime.Format "2006-01-02 15:04:05 UTC"}}

This sign-in was detected from a new device or location. If you recognize
this activity, you can safely ignore this message. If not, please review
your account and security settings.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`
  },
  {
    name: 'one-time-access',
    component: OneTimeAccessEmail,
    sampleProps: {
      logoURL: "LOGO_URL_PLACEHOLDER",
      appName: "APP_NAME_PLACEHOLDER",
      data: {
        code: "CODE_PLACEHOLDER",
        loginLink: "LOGIN_LINK_PLACEHOLDER",
        loginLinkWithCode: "LOGIN_LINK_WITH_CODE_PLACEHOLDER",
        expirationString: "EXPIRATION_STRING_PLACEHOLDER",
      }
    },
    replacements: [
      { search: /LOGO_URL_PLACEHOLDER/g, replace: '{{.LogoURL}}' },
      { search: /APP_NAME_PLACEHOLDER/g, replace: '{{.AppName}}' },
      { search: /CODE_PLACEHOLDER/g, replace: '{{.Data.Code}}' },
      { search: /LOGIN_LINK_PLACEHOLDER/g, replace: '{{.Data.LoginLink}}' },
      { search: /LOGIN_LINK_WITH_CODE_PLACEHOLDER/g, replace: '{{.Data.LoginLinkWithCode}}' },
      { search: /EXPIRATION_STRING_PLACEHOLDER/g, replace: '{{.Data.ExpirationString}}' },
    ],
    conditionals: [],
    textTemplate: `{{ define "root" -}}
Login Code
====================

Click the link below to sign in to {{ .AppName }} with a login code. This code expires in {{.Data.ExpirationString}}.

{{ .Data.LoginLinkWithCode }}

Or visit {{ .Data.LoginLink }} and enter the the code "{{ .Data.Code }}".

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`
  },
  {
    name: 'api-key-expiring-soon',
    component: ApiKeyExpiringEmail,
    sampleProps: {
      logoURL: "LOGO_URL_PLACEHOLDER",
      appName: "APP_NAME_PLACEHOLDER",
      data: {
        name: "NAME_PLACEHOLDER",
        apiKeyName: "API_KEY_NAME_PLACEHOLDER",
        expiresAt: "EXPIRES_AT_PLACEHOLDER",
      }
    },
    replacements: [
      { search: /LOGO_URL_PLACEHOLDER/g, replace: '{{.LogoURL}}' },
      { search: /APP_NAME_PLACEHOLDER/g, replace: '{{.AppName}}' },
      { search: /NAME_PLACEHOLDER/g, replace: '{{.Data.Name}}' },
      { search: /API_KEY_NAME_PLACEHOLDER/g, replace: '{{.Data.ApiKeyName}}' },
      { search: /EXPIRES_AT_PLACEHOLDER/g, replace: '{{.Data.ExpiresAt.Format "2006-01-02 15:04:05 MST"}}' },
    ],
    conditionals: [],
    textTemplate: `{{ define "root" -}}
API Key Expiring Soon
====================

Hello {{ .Data.Name }},

This is a reminder that your API key "{{ .Data.ApiKeyName }}" will expire on {{ .Data.ExpiresAt.Format "2006-01-02 15:04:05 MST" }}.

Please generate a new API key if you need continued access.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`
  },
  {
    name: 'test',
    component: TestEmail,
    sampleProps: {
      logoURL: "LOGO_URL_PLACEHOLDER",
      appName: "APP_NAME_PLACEHOLDER",
    },
    replacements: [
      { search: /LOGO_URL_PLACEHOLDER/g, replace: '{{.LogoURL}}' },
      { search: /APP_NAME_PLACEHOLDER/g, replace: '{{.AppName}}' },
    ],
    conditionals: [],
    textTemplate: `{{ define "root" -}}
This is a test email.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`
  }
] as const;

async function buildAllTemplates() {
  console.log('Building all email templates...');

  for (const template of templates) {
    console.log(`Building ${template.name}...`);
    
    try {
      const Component = template.component as any;
      const html = await render(Component(template.sampleProps));
      
      let goTemplateHtml = html;
      
      // Apply replacements BEFORE cleaning HTML comments
      for (const replacement of template.replacements) {
        goTemplateHtml = goTemplateHtml.replace(replacement.search, replacement.replace);
      }
      
      // Apply conditionals
      for (const conditional of template.conditionals) {
        const match = goTemplateHtml.match(conditional.regex);
        if (match) {
          const wrappedContent = conditional.wrapper.replace('$1', match[1]);
          goTemplateHtml = goTemplateHtml.replace(conditional.regex, wrappedContent);
        }
      }
      
      // Clean up React comments but KEEP MSO comments for Outlook compatibility
      goTemplateHtml = goTemplateHtml
        .replace(/<!--\$-->/g, '')
        .replace(/<!--\/\$-->/g, '')
        .replace(/<!--\d+-->/g, '')
        // Don't remove MSO comments - they're needed for Outlook
        .replace(/<!--(?!\[if mso\]|!\[endif\])[^>]*-->/g, '');
      
      const goHtmlTemplate = `{{define "root"}}${goTemplateHtml}{{end}}`;
      
      const htmlTemplatePath = path.join(outputDir, `${template.name}_html.tmpl`);
      fs.writeFileSync(htmlTemplatePath, goHtmlTemplate);
      
      const textTemplatePath = path.join(outputDir, `${template.name}_text.tmpl`);
      fs.writeFileSync(textTemplatePath, template.textTemplate);
      
      console.log(`✓ Generated ${htmlTemplatePath} and ${textTemplatePath}`);
    } catch (error) {
      console.error(`✗ Error building ${template.name}:`, error);
    }
  }
  
  console.log('All templates built successfully!');
}

async function main() {
  cleanupOldTemplates();
  await buildAllTemplates();
}

main().catch(console.error);