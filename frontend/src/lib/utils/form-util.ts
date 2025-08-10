import { get, writable } from 'svelte/store';
import { z } from 'zod/v4';

export type FormInput<T> = {
	value: T;
	error: string | null;
};

type FormInputs<T> = {
	[K in keyof T]: FormInput<T[K]>;
};

export function createForm<T extends z.ZodType<any, any>>(schema: T, initialValues: z.infer<T>) {
	// Create a writable store for the inputs
	const inputsStore = writable<FormInputs<z.infer<T>>>(initializeInputs(initialValues));
	const errorsStore = writable<z.ZodError<any> | undefined>();

	function initializeInputs(initialValues: z.infer<T>): FormInputs<z.infer<T>> {
		const inputs: FormInputs<z.infer<T>> = {} as FormInputs<z.infer<T>>;
		for (const key in initialValues) {
			if (Object.prototype.hasOwnProperty.call(initialValues, key)) {
				inputs[key as keyof z.infer<T>] = {
					value: initialValues[key as keyof z.infer<T>],
					error: null
				};
			}
		}
		return inputs;
	}

	function validate() {
		let success = true;
		inputsStore.update((inputs) => {
			// Extract values from inputs to validate against the schema
			const values = Object.fromEntries(
				Object.entries(inputs).map(([key, input]) => [key, input.value])
			);

			const result = schema.safeParse(values);
			errorsStore.set(result.error);

			if (!result.success) {
				success = false;
				for (const input of Object.keys(inputs)) {
					const error = result.error.issues.find((e) => e.path[0] === input);
					if (error) {
						inputs[input as keyof z.infer<T>].error = error.message;
					} else {
						inputs[input as keyof z.infer<T>].error = null;
					}
				}
			} else {
				for (const input of Object.keys(inputs)) {
					inputs[input as keyof z.infer<T>].error = null;
				}
			}
			// Update the input values with the parsed data
			for (const key in result.data) {
				if (Object.prototype.hasOwnProperty.call(inputs, key)) {
					inputs[key as keyof z.infer<T>].value = result.data[key];
				}
			}

			return inputs;
		});
		return success ? data() : null;
	}

	function data() {
		const inputs = get(inputsStore);

		const values = Object.fromEntries(
			Object.entries(inputs).map(([key, input]) => {
				input.value = trimValue(input.value);
				return [key, input.value];
			})
		) as z.infer<T>;

		return values;
	}

	function reset() {
		inputsStore.update((inputs) => {
			for (const input of Object.keys(inputs)) {
				inputs[input as keyof z.infer<T>] = {
					value: initialValues[input as keyof z.infer<T>],
					error: null
				};
			}
			return inputs;
		});
	}

	function setValue(key: keyof z.infer<T>, value: z.infer<T>[keyof z.infer<T>]) {
		inputsStore.update((inputs) => {
			inputs[key].value = value;
			return inputs;
		});
	}

	// Trims whitespace from string values and arrays of strings
	function trimValue(value: any) {
		if (typeof value === 'string') {
			value = value.trim();
		} else if (Array.isArray(value)) {
			value = value.map((item: any) => {
				if (typeof item === 'string') {
					return item.trim();
				}
				return item;
			});
		}
		return value;
	}

	return {
		schema,
		inputs: inputsStore,
		errors: errorsStore,
		data,
		validate,
		setValue,
		reset
	};
}
