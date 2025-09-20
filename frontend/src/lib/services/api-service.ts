import axios from 'axios';

abstract class APIService {
  protected api = axios.create({ baseURL: '/api' });

  constructor() {
    if (typeof process !== 'undefined' && process?.env?.DEVELOPMENT_BACKEND_URL) {
      this.api.defaults.baseURL = process.env.DEVELOPMENT_BACKEND_URL;
    }
  }
}

export default APIService;
