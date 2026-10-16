import {
    REMAINING_DISCHARGE_TIME_POINT_DATA_ID,
    THERMAL_RUNAWAY_RISK_POINT_DATA_ID,
    TEMPERATURE_DIFFERENCE_POINT_DATA_ID,
    PRESSURE_POINT_DATA_ID,
} from '../../utils/battery-point-ids.js';

export const TTDC_CID42_SOC_TEMPLATE_ROWS = [
    ...Array.from({ length: 16 }, (_, i) => {
        const n = i + 1;
        return {
            soc42Id: `alg_cell${String(n).padStart(2, '0')}_soc`,
            name: `${n}号单体SOC`,
            dataType: 1,
        };
    }),
    { soc42Id: REMAINING_DISCHARGE_TIME_POINT_DATA_ID, name: '放电时长', dataType: 1 },
    { soc42Id: 'alg_soc', name: 'SOC', dataType: 1 },
    { soc42Id: 'alg_soh', name: 'SOH', dataType: 1 },
    { soc42Id: TEMPERATURE_DIFFERENCE_POINT_DATA_ID, name: '温差', dataType: 1 },
    { soc42Id: PRESSURE_POINT_DATA_ID, name: '压差', dataType: 1 },
    { soc42Id: THERMAL_RUNAWAY_RISK_POINT_DATA_ID, name: '热失控风险等级', dataType: 2 },
];
export function getTtdcCid42SocAddressPrefix(ttdcAddress) {
    const address = String(ttdcAddress ?? '').trim();
    return address ? `${address}_` : '';
}
export function createTtdcCid42SocFormPoints(ttdcAddress) {
    const prefix = getTtdcCid42SocAddressPrefix(ttdcAddress);
    return TTDC_CID42_SOC_TEMPLATE_ROWS.map(row => ({
        name: `${prefix}${row.name}`,
        transport: '',
        dataType: row.dataType,
        formula: 'x',
        unformula: 'x',
        soc42Id: `${prefix}${row.soc42Id}`,
        __pt: { addr: null, bytes: null, convType: null, bit: null },
    }));
}
export function createTtdcCid42SocSubmitPoints(ttdcAddress) {
    const prefix = getTtdcCid42SocAddressPrefix(ttdcAddress);
    return TTDC_CID42_SOC_TEMPLATE_ROWS.map(row => ({
        name: `${prefix}${row.name}`,
        transport: '-',
        dataType: row.dataType,
        formula: 'x',
        unformula: 'x',
        dataId: `${prefix}${row.soc42Id}`,
    }));
}
