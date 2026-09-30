package replay

var tauntTexts = map[int]string{
	1: "Yes", 2: "No", 3: "Food, please", 4: "Wood, please", 5: "Gold, please",
	6: "Stone, please", 7: "Ahh", 8: "All hail", 9: "Oooh", 10: "Back to Age 1",
	11: "Laugh", 12: "Being rushed", 13: "Blame your isp", 14: "Start the game", 15: "Don't Point That Thing",
	16: "Enemy Sighted", 17: "It Is Good", 18: "I Need a Monk", 19: "Long Time No Siege", 20: "My granny",
	21: "Nice Town I'll Take It", 22: "Quit Touchin", 23: "Raiding Party", 24: "Dadgum", 25: "Smite Me",
	26: "The Wonder", 27: "You play 2 hours", 28: "You Should See the Other Guy", 29: "Roggan", 30: "Wololo",
	31: "Attack an Enemy Now", 32: "Cease Creating Extra Villagers", 33: "Create Extra Villagers", 34: "Build a Navy", 35: "Stop Building a Navy",
	36: "Wait for My Signal to Attack", 37: "Build a Wonder", 38: "Give Me Your Extra Resources", 39: "Ally", 40: "Enemy",
	41: "Neutral", 42: "What Age Are You In?", 43: "What is your strategy?", 44: "How many resources do you have?", 45: "Retreat now!",
	46: "Flare location of your army", 47: "Attack toward flared location", 48: "I'm being attacked, please help!", 49: "Build forward base at flare", 50: "Build fortification at flare",
	51: "Keep army close, fight with me", 52: "Build a market at flare", 53: "Rebuild your base at flare", 54: "Build wall between two flares", 55: "Build wall around your town",
	56: "Train counter units", 57: "Stop training counter units", 58: "Prepare to send me all resources", 59: "Stop sending me extra resources", 60: "Prepare to train large army",
	61: "Attack player 1!", 62: "Attack player 2!", 63: "Attack player 3!", 64: "Attack player 4!", 65: "Attack player 5!",
	66: "Attack player 6!", 67: "Attack player 7!", 68: "Attack player 8!", 69: "Delete object on flare", 70: "Delete excess villagers",
	71: "Delete excess warships", 72: "Focus on training infantry", 73: "Focus on training cavalry", 74: "Focus on training ranged", 75: "Focus on training warships",
	76: "Attack w/ Militia", 77: "Attack w/ Archers", 78: "Attack w/ Skirmishers", 79: "Attack w/ Archers+Skirms", 80: "Attack w/ Scout Cavalry",
	81: "Attack w/ Men-at-Arms", 82: "Attack w/ Eagle Scouts", 83: "Attack w/ Towers", 84: "Attack w/ Crossbowmen", 85: "Attack w/ Cavalry Archers",
	86: "Attack w/ Unique Units", 87: "Attack w/ Knights", 88: "Attack w/ Battle Elephants", 89: "Attack w/ Scorpions", 90: "Attack w/ Monks",
	91: "Attack w/ Monks+Mangonels", 92: "Attack w/ Eagle Warriors", 93: "Attack w/ Halberdiers+Rams", 94: "Attack w/ Elite Eagle Warriors", 95: "Attack w/ Arbalests",
	96: "Attack w/ Champions", 97: "Attack w/ Galleys", 98: "Attack w/ Fire Galleys", 99: "Attack w/ Demolition Rafts", 100: "Attack w/ War Galleys",
	101: "Attack w/ Fire Ships", 102: "Attack w/ Unique Warships", 103: "Onager cut trees at flare", 104: "Don't resign!", 105: "You can resign again",
}

func TauntText(number int) string {
	return tauntTexts[number]
}
