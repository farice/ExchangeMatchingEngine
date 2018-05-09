import sys

file_name = sys.argv[1]
function_name = sys.argv[2]

f = open(file_name)
lines = f.read().split("\n")
total = 0
count = 0
for l in lines:
	cols = l.split(",")
	if cols[0] == "request_handler.handleRequest":
		multiplier = 1
		num = 0
		if cols[1].endswith("ms"):
			pass
		elif cols[1].endswith("ns"):
			multiplier = (1/1000000)
		else:
			multiplier = (1/1000)

		for i in range(0, 3):
			try:
				num = float(cols[1][:len(cols[1])-i])
			except:
				continue

		total += num * multiplier
		count += 1

print(total/count)

