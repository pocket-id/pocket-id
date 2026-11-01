import { Heading, Text } from 'react-email';
import { colors, fonts } from './theme';

interface CardHeaderProps {
	title: string;
	children: React.ReactNode;
}

// Mirrors the app's AuthHeading: a display title with a larger muted subtitle that says what the email is about
export default function CardHeader({ title, children }: CardHeaderProps) {
	return (
		<>
			<Heading as="h1" style={titleStyle}>
				{title}
			</Heading>
			<Text style={subtitleStyle}>{children}</Text>
		</>
	);
}

const titleStyle = {
	margin: '0 0 8px 0',
	fontFamily: fonts.display,
	fontSize: '30px',
	lineHeight: '38px',
	fontWeight: 600,
	letterSpacing: '-0.02em',
	color: colors.fg
};

const subtitleStyle = {
	margin: '0 0 24px 0',
	fontSize: '17px',
	lineHeight: '26px',
	color: colors.fgMuted
};
