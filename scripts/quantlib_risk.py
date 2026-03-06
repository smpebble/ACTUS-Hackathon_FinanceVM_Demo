import sys
import json
import math

# We wrap the QuantLib import in a try-except just in case to provide fallback simulation 
# if installation failed or isn't fully set up in the environment.
try:
    import QuantLib as ql
    HAS_QUANTLIB = True
except ImportError:
    HAS_QUANTLIB = False

def safe_float(val):
    """Safely convert a value to float, handling strings and decimals."""
    try:
        if isinstance(val, str):
            return float(val)
        return float(val)
    except (ValueError, TypeError):
        return 0.0

def fallback_risk_analysis(contract_type, total_notional, cashflows):
    """
    Fallback mathematical approximation of risk if QuantLib is unavailable.
    """
    cf_variance = len(cashflows) * 0.0001
    base_market_risk = 0.0125
    base_credit_risk = 0.005
    base_liquidity = 0.002
    base_counterparty = 0.001
    base_stress = -0.045
    
    if contract_type == "ANN":
        base_market_risk = 0.0210
        base_credit_risk = 0.015
        base_stress = -0.065
    elif contract_type == "SWAPS":
        base_market_risk = 0.0350
        base_counterparty = 0.025
        base_stress = -0.080
    elif contract_type == "PAM" and len(cashflows) <= 3: # stablecoin
        base_market_risk = 0.0001
        base_stress = -0.01
        
    # Introduce deterministic pseudo-randomness using the total_notional and length
    deterministic_seed = int(total_notional) % 1000 + len(cashflows)
    
    market_risk = total_notional * (base_market_risk + cf_variance + (deterministic_seed % 10)*0.0001)
    credit_risk = total_notional * (base_credit_risk + cf_variance/2 + (deterministic_seed % 7)*0.0001)
    liquidity_risk = total_notional * (base_liquidity + cf_variance/3 + (deterministic_seed % 5)*0.0001)
    counterparty_risk = total_notional * (base_counterparty + cf_variance/4 + (deterministic_seed % 3)*0.0001)
    stress_test = total_notional * (base_stress - cf_variance * 5 - (deterministic_seed % 11)*0.001)
    
    return {
        "marketRisk": str(round(market_risk, 5)),
        "creditRisk": str(round(credit_risk, 5)),
        "liquidityRisk": str(round(liquidity_risk, 5)),
        "counterpartyRisk": str(round(counterparty_risk, 5)),
        "stressTestImpact": str(round(stress_test, 5)),
        "engine": "Dynamic Mathematical Fallback"
    }

def quantlib_risk_analysis(contract_type, notional, cashflows, rate):
    """
    Utilize QuantLib to generate more realistic risk metrics.
    """
    try:
        # 1. Setup the evaluation date to the first cashflow or today
        today = ql.Date.todaysDate()
        ql.Settings.instance().evaluationDate = today
        
        # 2. Yield Curve Setup (Flat Forward for simplicity)
        forecast_curve = ql.YieldTermStructureHandle(
            ql.FlatForward(today, rate if rate > 0 else 0.05, ql.Actual365Fixed())
        )
        
        # 3. Simulate Volatility Surface based on contract type
        vol = 0.15 # Default 15% volatility
        if contract_type == "SWAPS":
            vol = 0.25
        elif contract_type == "ANN":
            vol = 0.18
        elif contract_type == "PAM" and len(cashflows) <= 3:
            vol = 0.01 # Stablecoin
        
        # We construct a mock bond to price the cashflows
        schedule_dates = []
        amounts = []
        
        for cf in cashflows:
            # Parse time - Go marshals time.Time as "2026-03-04T00:00:00Z" or similar ISO format
            time_str = str(cf.get("time", ""))
            if "T" in time_str:
                date_str = time_str.split("T")[0]
                parts = date_str.split("-")
                if len(parts) == 3:
                    try:
                        year = int(parts[0])
                        month = int(parts[1])
                        day = int(parts[2])
                        d = ql.Date(day, month, year)
                        # Only add future dates
                        if d > today:
                            schedule_dates.append(d)
                            amounts.append(safe_float(cf.get("payoff", 0)))
                    except Exception:
                        continue
                    
        # Use standard QuantLib NPV as a base for stress testing
        if len(schedule_dates) > 0 and len(amounts) > 0:
            npv = sum(amount * forecast_curve.discount(d) for amount, d in zip(amounts, schedule_dates))
        else:
            npv = notional
        
        if abs(npv) < 0.01:
            npv = notional
            
        # Calculate Value at Risk (VaR) mathematically using standard deviation
        # 99% VaR assuming normal distribution
        z_score = 2.326 
        var_99 = abs(npv) * vol * z_score * math.sqrt(1.0/252.0) # 1-day 99% VaR
        
        market_risk = var_99
        
        # Credit Risk (Expected Credit Loss)
        # Default Probability modeled roughly
        pd = 0.02
        lgd = 0.45
        if contract_type == "SWAPS":
            pd = 0.005 # Bank counterparty usually
        elif contract_type == "ANN":
            pd = 0.05 # SME loan
        elif contract_type == "PAM" and len(cashflows) <= 3:
            pd = 0.001
        
        credit_risk = abs(npv) * pd * lgd
        
        # Liquidity Risk (Bid/Ask spread simulation)
        liquidity_spread = 0.001 if contract_type == "PAM" else (0.005 if contract_type == "SWAPS" else 0.02)
        liquidity_risk = notional * liquidity_spread
        
        # Counterparty Risk (CVA approximation)
        if contract_type == "SWAPS":
            counterparty_risk = credit_risk * 0.5
        else:
            counterparty_risk = credit_risk * 0.1
        
        # Stress Test Impact (Parallel shift in yield curve + 200 bps)
        stress_curve = ql.YieldTermStructureHandle(
            ql.FlatForward(today, (rate if rate > 0 else 0.05) + 0.02, ql.Actual365Fixed())
        )
        if len(schedule_dates) > 0 and len(amounts) > 0:
            stress_npv = sum(amount * stress_curve.discount(d) for amount, d in zip(amounts, schedule_dates))
        else:
            stress_npv = npv * 0.98
        
        stress_impact = stress_npv - npv
        
        # If the calculations yield zero or very low numbers, fallback to a small % of notional
        if market_risk < 0.01:
            market_risk = notional * vol * 0.01
        
        return {
            "marketRisk": str(round(market_risk, 5)),
            "creditRisk": str(round(credit_risk, 5)),
            "liquidityRisk": str(round(liquidity_risk, 5)),
            "counterpartyRisk": str(round(counterparty_risk, 5)),
            "stressTestImpact": str(round(stress_impact, 5)),
            "engine": "QuantLib"
        }
    except Exception as e:
        # If QuantLib analysis fails, fall back to mathematical method
        return fallback_risk_analysis(contract_type, notional, cashflows)

def quantlib_risk_analysis_dynamic(contract_type, notional, cashflows, rate, vol_override=None, z_score_override=None, stress_shift_bps=200, pd_override=None, lgd_override=None):
    """
    Dynamic QuantLib risk analysis that accepts custom risk parameters from the frontend.
    """
    try:
        today = ql.Date.todaysDate()
        ql.Settings.instance().evaluationDate = today

        forecast_curve = ql.YieldTermStructureHandle(
            ql.FlatForward(today, rate if rate > 0 else 0.05, ql.Actual365Fixed())
        )

        # Use provided vol or default based on contract type
        if vol_override is not None and vol_override > 0:
            vol = vol_override
        else:
            vol = 0.15
            if contract_type == "SWAPS": vol = 0.25
            elif contract_type == "ANN": vol = 0.18
            elif contract_type == "PAM" and len(cashflows) <= 3: vol = 0.01

        # Z-score from confidence level
        z_score = z_score_override if z_score_override and z_score_override > 0 else 2.326

        # PD override
        if pd_override is not None and pd_override >= 0:
            pd = pd_override
        else:
            pd = 0.02
            if contract_type == "SWAPS": pd = 0.005
            elif contract_type == "ANN": pd = 0.05
            elif contract_type == "PAM" and len(cashflows) <= 3: pd = 0.001

        # LGD override
        lgd = lgd_override if lgd_override is not None and lgd_override > 0 else 0.45

        # Parse cashflow dates
        schedule_dates, amounts = [], []
        for cf in cashflows:
            time_str = str(cf.get("time", ""))
            if "T" in time_str:
                parts = time_str.split("T")[0].split("-")
                if len(parts) == 3:
                    try:
                        d = ql.Date(int(parts[2]), int(parts[1]), int(parts[0]))
                        if d > today:
                            schedule_dates.append(d)
                            amounts.append(safe_float(cf.get("payoff", 0)))
                    except Exception:
                        continue

        npv = sum(amount * forecast_curve.discount(d) for amount, d in zip(amounts, schedule_dates)) if schedule_dates else notional
        if abs(npv) < 0.01: npv = notional

        # Market Risk (VaR) using dynamic Z-score
        var = abs(npv) * vol * z_score * math.sqrt(1.0/252.0)
        market_risk = max(var, notional * vol * 0.01)

        # Credit Risk (ECL = NPV × PD × LGD)
        credit_risk = abs(npv) * pd * lgd

        # Liquidity Risk
        liquidity_spread = 0.001 if contract_type == "PAM" else (0.005 if contract_type == "SWAPS" else 0.02)
        liquidity_risk = notional * liquidity_spread

        # Counterparty Risk (CVA)
        counterparty_risk = credit_risk * 0.5 if contract_type == "SWAPS" else credit_risk * 0.1

        # Stress Test: parallel yield curve shift by stress_shift_bps
        stress_rate = (rate if rate > 0 else 0.05) + (stress_shift_bps / 10000.0)
        stress_curve = ql.YieldTermStructureHandle(ql.FlatForward(today, stress_rate, ql.Actual365Fixed()))
        stress_npv = sum(amount * stress_curve.discount(d) for amount, d in zip(amounts, schedule_dates)) if schedule_dates else npv * (1 - stress_shift_bps / 10000.0)
        stress_impact = stress_npv - npv

        return {
            "marketRisk": str(round(market_risk, 5)),
            "creditRisk": str(round(credit_risk, 5)),
            "liquidityRisk": str(round(liquidity_risk, 5)),
            "counterpartyRisk": str(round(counterparty_risk, 5)),
            "stressTestImpact": str(round(stress_impact, 5)),
            "engine": f"QuantLib (σ={vol*100:.1f}% Z={z_score} PD={pd*100:.2f}% Δr={stress_shift_bps}bps)"
        }
    except Exception as e:
        return fallback_risk_analysis(contract_type, notional, cashflows)


def main():
    try:
        input_data = sys.stdin.read()
        if not input_data or not input_data.strip():
            print(json.dumps(fallback_risk_analysis("PAM", 1000000, [])))
            return

        data = json.loads(input_data)

        contract_type = data.get("contractType", "PAM")
        notional = safe_float(data.get("notional", "1000000"))
        rate = safe_float(data.get("rate", "0.05"))
        if notional <= 0: notional = 1000000
        if rate <= 0: rate = 0.05

        cashflows = data.get("events", [])

        # --- Read dynamic parameters sent from the frontend sliders ---
        params = data.get("params", {})
        vol_override      = safe_float(params.get("vol", None)) if params.get("vol") else None
        z_score_override  = safe_float(params.get("zScore", None)) if params.get("zScore") else None
        stress_shift_bps  = int(safe_float(params.get("stressShiftBps", "200")))
        pd_override       = safe_float(params.get("pd", None)) if params.get("pd") else None
        lgd_override      = safe_float(params.get("lgd", None)) if params.get("lgd") else None

        has_dynamic_params = any([vol_override, z_score_override, pd_override, lgd_override])

        if HAS_QUANTLIB and len(cashflows) > 0:
            if has_dynamic_params:
                result = quantlib_risk_analysis_dynamic(
                    contract_type, notional, cashflows, rate,
                    vol_override=vol_override, z_score_override=z_score_override,
                    stress_shift_bps=stress_shift_bps, pd_override=pd_override, lgd_override=lgd_override
                )
            else:
                result = quantlib_risk_analysis(contract_type, notional, cashflows, rate)
        else:
            result = fallback_risk_analysis(contract_type, notional, cashflows)

        print(json.dumps(result))

    except Exception as e:
        print(json.dumps({
            "marketRisk": "100", "creditRisk": "50",
            "liquidityRisk": "20", "counterpartyRisk": "10",
            "stressTestImpact": "-500",
            "engine": "Error Fallback: " + str(e)
        }))


if __name__ == "__main__":
    main()

