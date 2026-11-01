import { Button as EmailButton, Section } from 'react-email';
import { colors, fonts, radius } from './theme';

interface ButtonProps {
	href: string;
	children: React.ReactNode;
}

// The full-width primary of an auth step in the app
export const Button = ({ href, children }: ButtonProps) => (
	<Section style={containerStyle}>
		<EmailButton href={href} style={buttonStyle}>
			{children}
		</EmailButton>
	</Section>
);

const containerStyle = {
	margin: '24px 0 0 0'
};

const buttonStyle = {
	display: 'block',
	padding: '16px 24px',
	backgroundColor: colors.primary,
	color: colors.primaryForeground,
	borderRadius: radius.control,
	fontFamily: fonts.sans,
	fontSize: '15px',
	lineHeight: '20px',
	fontWeight: 600,
	textAlign: 'center' as const,
	textDecoration: 'none'
};
