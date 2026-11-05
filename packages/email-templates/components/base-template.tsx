import { Body, Column, Container, Head, Html, Img, Link, Preview, Row, Section } from 'react-email';
import type { SharedProps } from '../props';
import { colors, fonts, radius } from './theme';

interface BaseTemplateProps extends SharedProps {
	preview?: string;
	children: React.ReactNode;
}

// Every email is the sign-in card of the app: the logo tile, a display heading and one filled action on a white card
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
			<style dangerouslySetInnerHTML={{ __html: headCss(appURL) }} />
		</Head>
		{preview && <Preview>{preview}</Preview>}
		<Body style={bodyStyle}>
			<Container style={containerStyle}>
				<Section>
					<Row>
						<Column className="card" style={cardStyle}>
							<Section style={tileStyle} data-skip-in-text="true">
								<Row>
									<Column style={tileCellStyle}>
										<Link href={appURL}>
											<Img src={logoURL} width="32" height="32" alt={appName} style={logoStyle} />
										</Link>
									</Column>
								</Row>
							</Section>
							{children}
						</Column>
					</Row>
				</Section>
			</Container>
		</Body>
	</Html>
);

// The fonts are self-hosted by the app, so they load from the same origin as the logo and work without third-party requests
// Clients without web font support fall back to the system stacks in the theme
// Phones get the tighter padding of the app's sheet, because the desktop card's padding would leave too little room for the code
const headCss = (appURL: string) =>
	[
		`@font-face{font-family:'Young Serif';font-style:normal;font-weight:400;mso-font-alt:'Georgia';src:url(${appURL}/fonts/YoungSerif-latin.woff2) format('woff2');}`,
		`@font-face{font-family:'DM Sans';font-style:normal;font-weight:100 1000;mso-font-alt:'Arial';src:url(${appURL}/fonts/DMSans-latin.woff2) format('woff2');}`,
		`@font-face{font-family:'JetBrains Mono';font-style:normal;font-weight:400 500;mso-font-alt:'Courier New';src:url(${appURL}/fonts/JetBrainsMono-latin.woff2) format('woff2');}`,
		`@media (max-width:520px){.card{padding:32px 24px !important;}}`
	].join('');

const bodyStyle = {
	margin: 0,
	padding: '40px 12px',
	backgroundColor: colors.canvas,
	fontFamily: fonts.sans,
	color: colors.fg
};

const containerStyle = {
	width: '100%',
	maxWidth: '520px',
	margin: '0 auto'
};

const cardStyle = {
	backgroundColor: colors.surface,
	border: `1px solid ${colors.border}`,
	borderRadius: radius.card,
	padding: '48px',
	textAlign: 'left' as const
};

const tileStyle = {
	width: '56px',
	margin: '0 0 24px 0',
	border: `1px solid ${colors.border}`,
	borderRadius: radius.tile
};

const tileCellStyle = {
	width: '54px',
	height: '54px',
	textAlign: 'center' as const,
	verticalAlign: 'middle'
};

const logoStyle = {
	display: 'inline-block',
	width: '32px',
	height: '32px'
};
