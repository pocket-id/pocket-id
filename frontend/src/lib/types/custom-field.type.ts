export type CustomFieldValue = {
	customFieldId: string;
	value: string;
};

export type CustomFieldType = 'string' | 'number' | 'boolean';
export type CustomFieldTarget = 'user' | 'group' | 'both';

export type CustomField = {
	id: string;
	key: string;
	displayName: string;
	type: CustomFieldType;
	target: CustomFieldTarget;
	required: boolean;
	userEditable: boolean;
	defaultValue?: string;
	validationRegex?: string;
	validationErrorMessage?: string;
};

export function customFieldAppliesTo(field: CustomField, target: 'user' | 'group') {
	return field.target === 'both' || field.target === target;
}
