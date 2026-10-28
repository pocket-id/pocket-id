import type { Features } from '#lib/types/feature.type.ts';
import APIService from './api-service';

export default class FeatureService extends APIService {
	list = async () => (await this.api.get<Features>('/features')).data;
}
