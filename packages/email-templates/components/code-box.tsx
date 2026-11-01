import { Column, Row, Section, Text } from 'react-email';
import { colors, fonts, radius } from './theme';

interface CodeBoxProps {
	code: string;
}

// One cell instead of the app's per-character cells, so copying the code from the email never picks up separators
export const CodeBox = ({ code }: CodeBoxProps) => (
	<Section style={boxStyle}>
		<Row>
			<Column style={cellStyle}>
				<Text style={codeStyle}>{code}</Text>
			</Column>
		</Row>
	</Section>
);

const boxStyle = {
	backgroundColor: colors.well,
	border: `1px solid ${colors.border}`,
	borderRadius: radius.tile
};

const cellStyle = {
	padding: '17px 16px',
	textAlign: 'center' as const
};

const codeStyle = {
	margin: 0,
	fontFamily: fonts.mono,
	fontSize: '30px',
	lineHeight: '38px',
	fontWeight: 500,
	letterSpacing: '0.3em',
	paddingLeft: '0.3em',
	color: colors.fg
};
