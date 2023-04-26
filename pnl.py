import json

total_pnl = 0
entry = 0

win_count = 0
total_count = 0

prev_cost = 0
total_cost = 0

total_trading_volume = 0

# 0 means no position, 1 means long, -1 means short
position = 0

# Read the JSON file
with open('orders.json', 'r') as f:
    # Load JSON objects from each line
    for line in f:
        try:
            data = json.loads(line)
            # Access the individual fields as needed
            side = data['side']
            qty = float(data['qty'])
            price = float(data['price'])
            cost = float(data['cost'])

            total_cost += cost
            total_trading_volume += price * qty

            
            if position == 0:
                # no opened position
                if side == "0":
                    position = 1
                else:
                    position = -1
                
                prev_cost = cost
                entry = price
                
            else:
                current_pnl = position * ( price - entry ) * qty - prev_cost - cost

                print(f"PnL($): {current_pnl}")
                
                total_count += 1
                if current_pnl > 0:
                    win_count += 1
                total_pnl += current_pnl

                position = 0
        except json.JSONDecodeError as e:
            print(f"Error decoding JSON: {e}")
    
    print()
    print(f"Total PnL($): {total_pnl}")
    print(f"Total Cost($): {total_cost}")
    print(f"Won {win_count} out of {total_count} trades. win rate(%): {win_count/total_count*100}%")
    print(f"Volume($): {total_trading_volume}")
