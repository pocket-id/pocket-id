import { render } from '@react-email/components';
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

// Auto-generate placeholders based on template name
function getSampleProps(templateName: string) {
  const baseProps = {
    logoURL: "LOGOURL_PLACEHOLDER",
    appName: "APPNAME_PLACEHOLDER",
  };

  const dataMap: Record<string, any> = {
    'new-signin': {
      city: "CITY_PLACEHOLDER",
      country: "COUNTRY_PLACEHOLDER",
      ipAddress: "IPADDRESS_PLACEHOLDER",
      device: "DEVICE_PLACEHOLDER",
      dateTime: "DATETIME_PLACEHOLDER",
    },
    'one-time-access': {
      code: "CODE_PLACEHOLDER",
      loginLink: "LOGINLINK_PLACEHOLDER",
      buttonCodeLink: "BUTTONCODELINK_PLACEHOLDER",
      expirationString: "EXPIRATIONSTRING_PLACEHOLDER",
    },
    'api-key-expiring': {
      name: "NAME_PLACEHOLDER",
      apiKeyName: "APIKEYNAME_PLACEHOLDER",
      expiresAt: "EXPIRESAT_PLACEHOLDER",
    },
    'test': {},
  };

  const data = dataMap[templateName] || {};
  
  return Object.keys(data).length > 0 
    ? { ...baseProps, data } 
    : baseProps;
}

// Auto-generate Go template replacements
function getReplacements(templateName: string) {
  const baseReplacements = [
    { search: /LOGOURL_PLACEHOLDER/g, replace: '{{.LogoURL}}' },
    { search: /APPNAME_PLACEHOLDER/g, replace: '{{.AppName}}' },
  ];

  const dataReplacements: Record<string, Array<{ search: RegExp; replace: string }>> = {
    'new-signin': [
      { search: /CITY_PLACEHOLDER/g, replace: '{{.Data.City}}' },
      { search: /COUNTRY_PLACEHOLDER/g, replace: '{{.Data.Country}}' },
      { search: /IPADDRESS_PLACEHOLDER/g, replace: '{{.Data.IPAddress}}' },
      { search: /DEVICE_PLACEHOLDER/g, replace: '{{.Data.Device}}' },
      { search: /DATETIME_PLACEHOLDER/g, replace: '{{.Data.DateTime.Format "January 2, 2006 at 3:04 PM MST"}}' },
    ],
    'one-time-access': [
      { search: /CODE_PLACEHOLDER/g, replace: '{{.Data.Code}}' },
      { search: /LOGINLINK_PLACEHOLDER/g, replace: '{{.Data.LoginLink}}' },
      { search: /BUTTONCODELINK_PLACEHOLDER/g, replace: '{{.Data.LoginLinkWithCode}}' },
      { search: /EXPIRATIONSTRING_PLACEHOLDER/g, replace: '{{.Data.ExpirationString}}' },
    ],
    'api-key-expiring': [
      { search: /NAME_PLACEHOLDER/g, replace: '{{.Data.Name}}' },
      { search: /APIKEYNAME_PLACEHOLDER/g, replace: '{{.Data.ApiKeyName}}' },
      { search: /EXPIRESAT_PLACEHOLDER/g, replace: '{{.Data.ExpiresAt.Format "2006-01-02 15:04:05 MST"}}' },
    ],
    'test': [],
  };

  return [...baseReplacements, ...(dataReplacements[templateName] || [])];
}

// Auto-generate text templates
function getTextTemplate(templateName: string) {
  const title = templateName
    .split('-')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');

  const templates: Record<string, string> = {
    'new-signin': `{{ define "root" -}}
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
{{ end -}}`,

    'one-time-access': `{{ define "root" -}}
Login Code
====================

Click the link below to sign in to {{ .AppName }} with a login code. This code expires in {{.Data.ExpirationString}}.

{{ .Data.LoginLinkWithCode }}

Or visit {{ .Data.LoginLink }} and enter the the code "{{ .Data.Code }}".

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`,

    'api-key-expiring': `{{ define "root" -}}
API Key Expiring Soon
====================

Hello {{ .Data.Name }},

This is a reminder that your API key "{{ .Data.ApiKeyName }}" will expire on {{ .Data.ExpiresAt.Format "2006-01-02 15:04:05 MST" }}.

Please generate a new API key if you need continued access.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`,

    'test': `{{ define "root" -}}
This is a test email.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`,
  };

  return templates[templateName] || `{{ define "root" -}}
${title}
${'='.repeat(title.length)}

Email content goes here.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`;
}

// Apply special conditionals for specific templates
function applyConditionals(templateName: string, html: string): string {
  if (templateName === 'new-signin') {
    // Wrap location data in conditional
    const locationRegex = /(<div[^>]*>[\s\S]*?<.*?>Approximate Location<[\s\S]*?{{\.Data\.City}}, {{\.Data\.Country}}[\s\S]*?<\/div>)/;
    const match = html.match(locationRegex);
    if (match) {
      return html.replace(locationRegex, `{{if and .Data.City .Data.Country}}${match[1]}{{end}}`);
    }
  }
  return html;
}

// Convert template filename to template name
function getTemplateName(filename: string): string {
  return filename
    .replace('.tsx', '')
    .replace('-email', '');
}

async function discoverAndBuildTemplates() {
  console.log('Discovering and building email templates...');
  
  const emailsDir = './emails';
  const files = fs.readdirSync(emailsDir);
  
  for (const file of files) {
    // Skip base template and components
    if (file.endsWith('.tsx') && 
        !file.includes('base-template') && 
        !file.includes('components')) {
      
      const templateName = getTemplateName(file);
      const modulePath = path.join(emailsDir, file);
      
      console.log(`Building ${templateName}...`);
      
      try {
        // Dynamic import
        const module = await import(modulePath);
        const Component = module.default || module[Object.keys(module)[0]];
        
        if (!Component) {
          console.error(`✗ No component found in ${file}`);
          continue;
        }
        
        // Get sample props and render
        const sampleProps = getSampleProps(templateName);
        const html = await render(Component(sampleProps));
        
        // Apply replacements
        const replacements = getReplacements(templateName);
        let goTemplateHtml = html;
        
        for (const replacement of replacements) {
          goTemplateHtml = goTemplateHtml.replace(replacement.search, replacement.replace);
        }
        
        // Apply conditionals
        goTemplateHtml = applyConditionals(templateName, goTemplateHtml);
        
        // Convert template name for Go backend
        const goTemplateName = templateName === 'new-signin' ? 'login-with-new-device' : templateName;
        
        // Write HTML template
        const goHtmlTemplate = `{{define "root"}}${goTemplateHtml}{{end}}`;
        const htmlTemplatePath = path.join(outputDir, `${goTemplateName}_html.tmpl`);
        fs.writeFileSync(htmlTemplatePath, goHtmlTemplate);
        
        // Write text template
        const textTemplate = getTextTemplate(templateName);
        const textTemplatePath = path.join(outputDir, `${goTemplateName}_text.tmpl`);
        fs.writeFileSync(textTemplatePath, textTemplate);
        
        console.log(`✓ Generated ${htmlTemplatePath} and ${textTemplatePath}`);
      } catch (error) {
        console.error(`✗ Error building ${templateName}:`, error);
      }
    }
  }
}

async function main() {
  cleanupOldTemplates();
  await discoverAndBuildTemplates();
  console.log('All templates built successfully!');
}

main().catch(console.error);