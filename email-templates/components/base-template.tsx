import { Body, Column, Container, Head, Html, Img, Link, Row, Section } from 'react-email';
import type { SharedProps } from '../props';
import { colors, fonts, radius } from './theme';

interface BaseTemplateProps extends SharedProps {
	preview?: string;
	children: React.ReactNode;
}

export const BaseTemplate = ({
	logoURL,
	appName,
	appURL,
	preview,
	children
}: BaseTemplateProps) => (
	<Html lang="en">
		<Head>
			<meta name="color-scheme" content="light" />
			<meta name="supported-color-schemes" content="light" />
			{preview && <title>{preview}</title>}
			<style dangerouslySetInnerHTML={{ __html: fontFaceCss(appURL) }} />
		</Head>
		{preview && (
			// react-email's Preview pads the preheader with invisible characters; spam filters reject the message for it
			<div style={previewStyle} data-skip-in-text="true">
				{preview}
			</div>
		)}
		<Body style={bodyStyle}>
			<Container style={containerStyle}>
				<Section style={headerStyle} data-skip-in-text="true">
					<Row>
						<Column style={logoColumnStyle}>
							<Link href={appURL}>
								<Img src={logoURL} width="28" height="28" alt={appName} style={logoStyle} />
							</Link>
						</Column>
						<Column>
							<Link href={appURL} style={appNameStyle}>
								{appName}
							</Link>
						</Column>
					</Row>
				</Section>

				<Section>
					<Row>
						<Column style={cardStyle}>{children}</Column>
					</Row>
				</Section>
			</Container>
		</Body>
	</Html>
);

// The heading font is self-hosted by the app, so it loads from the same origin as the logo and works without third-party requests
// Clients without web font support fall back to the serif stack declared on the headings
const fontFaceCss = (appURL: string) =>
	`@font-face{font-family:'Gloock';font-style:normal;font-weight:400;mso-font-alt:'Georgia';src:url(${appURL}/fonts/Gloock-Regular.woff) format('woff');}`;

const previewStyle = {
	display: 'none',
	overflow: 'hidden',
	lineHeight: '1px',
	opacity: 0,
	maxHeight: 0,
	maxWidth: 0
};

const bodyStyle = {
	margin: 0,
	padding: '32px 16px',
	backgroundColor: colors.background,
	fontFamily: fonts.sans,
	color: colors.text
};

const containerStyle = {
	width: '100%',
	maxWidth: '480px',
	margin: '0 auto'
};

const headerStyle = {
	marginBottom: '20px'
};

const logoColumnStyle = {
	width: '36px',
	verticalAlign: 'middle'
};

const logoStyle = {
	display: 'block',
	width: '28px',
	height: '28px',
	borderRadius: '6px'
};

const appNameStyle = {
	fontFamily: fonts.serif,
	fontSize: '20px',
	lineHeight: '28px',
	color: colors.foreground,
	textDecoration: 'none'
};

const cardStyle = {
	backgroundColor: colors.card,
	border: `1px solid ${colors.border}`,
	borderRadius: radius.card,
	padding: '32px',
	textAlign: 'left' as const
};
