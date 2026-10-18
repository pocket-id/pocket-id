import Root, { filledFieldClass, readOnlyFieldClass } from './input.svelte';

export type FormInputEvent<T extends Event = Event> = T & {
	currentTarget: EventTarget & HTMLInputElement;
};

export {
	Root,
	filledFieldClass,
	readOnlyFieldClass,
	//
	Root as Input
};
