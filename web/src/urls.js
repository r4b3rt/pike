let prefix = window.location.pathname;
// 如果pathname是/，则使用默认前缀/pike/
if (prefix === '/') {
  prefix = '/pike/';
}

export const CONFIGS = `${prefix}configs/:category`;
export const CONFIG = `${prefix}configs/:category/:name`;
