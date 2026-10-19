export const TTDC_CONV_HEX_TO_SIGNED_DEC = 1;
export const TTDC_CONV_HEX_TO_BIN = 2;
export const TTDC_CONV_HEX_TO_UNSIGNED_DEC = 3;
export function isTtdcDecimalConversion(value) {
    const type = Number(value);
    return type === TTDC_CONV_HEX_TO_SIGNED_DEC || type === TTDC_CONV_HEX_TO_UNSIGNED_DEC;
}
export function formatTtdcConversionType(value) {
    const type = Number(value);
    if (type === TTDC_CONV_HEX_TO_SIGNED_DEC)
        return '十六进制转有符号十进制';
    if (type === TTDC_CONV_HEX_TO_BIN)
        return '十六进制转二进制';
    if (type === TTDC_CONV_HEX_TO_UNSIGNED_DEC)
        return '十六进制转无符号十进制';
    return '-';
}
