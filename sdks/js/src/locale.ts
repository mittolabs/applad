import type { Applad } from './client';

export class Locale {
  constructor(private client: Applad) {}

  listCountries() { return this.client.call('GET', '/locale/countries'); }
  listContinents() { return this.client.call('GET', '/locale/continents'); }
  listCurrencies() { return this.client.call('GET', '/locale/currencies'); }
  listLanguages() { return this.client.call('GET', '/locale/languages'); }
  get() { return this.client.call('GET', '/locale'); }
}
