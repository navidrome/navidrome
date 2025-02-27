package str_test

import (
	"github.com/navidrome/navidrome/utils/str"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("String Utils", func() {
	Describe("Clear", func() {
		DescribeTable("replaces some Unicode chars with their equivalent ASCII",
			func(input, expected string) {
				Expect(str.Clear(input)).To(Equal(expected))
			},
			Entry("k-os", "k–os", "k-os"),
			Entry("k‐os", "k‐os", "k-os"),
			Entry(`"Weird" Al Yankovic`, "“Weird” Al Yankovic", `"Weird" Al Yankovic`),
			Entry("Single quotes", "‘Single’ quotes", "'Single' quotes"),
		)
	})

	Describe("LongestCommonPrefix", func() {
		It("finds the longest common prefix", func() {
			Expect(str.LongestCommonPrefix(testPaths)).To(Equal("/Music/iTunes 1/iTunes Media/Music/"))
		})
		It("does NOT handle partial prefixes", func() {
			albums := []string{
				"/artist/albumOne",
				"/artist/albumTwo",
			}
			Expect(str.LongestCommonPrefix(albums)).To(Equal("/artist/album"))
		})
	})
})

var testPaths = []string{
	"/Music/iTunes 1/iTunes Media/Music/ABBA/Gold_ Greatest Hits/Dancing Queen.m4a",
	"/Music/iTunes 1/iTunes Media/Music/ABBA/Gold_ Greatest Hits/Mamma Mia.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Art Blakey/A Night At Birdland, Vol. 1/01 Annoucement By Pee Wee Marquette.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Art Blakey/A Night At Birdland, Vol. 1/02 Split Kick.m4a",
	"/Music/iTunes 1/iTunes Media/Music/As Frenéticas/As Frenéticas/Perigosa.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Bachman-Turner Overdrive/Gold/Down Down.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Bachman-Turner Overdrive/Gold/Hey You.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Bachman-Turner Overdrive/Gold/Hold Back The Water.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Belle And Sebastian/Write About Love/01 I Didn't See It Coming.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Belle And Sebastian/Write About Love/02 Come On Sister.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Black Eyed Peas/Elephunk/03 Let's Get Retarded.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Black Eyed Peas/Elephunk/04 Hey Mama.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Black Eyed Peas/Monkey Business/10 They Don't Want Music (Feat. James Brown).m4a",
	"/Music/iTunes 1/iTunes Media/Music/Black Eyed Peas/The E.N.D/1-01 Boom Boom Pow.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Black Eyed Peas/Timeless/01 Mas Que Nada.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Blondie/Heart Of Glass/Heart Of Glass.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Bob Dylan/Nashville Skyline/06 Lay Lady Lay.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Botany/Feeling Today - EP/03 Waterparker.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Céu/CéU/06 10 Contados.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Chance/Six Through Ten/03 Forgive+Forget.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Clive Tanaka Y Su Orquesta/Jet Set Siempre 1°/03 Neu Chicago (Side A) [For Dance].m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Absolute Rock Classics/1-02 Smoke on the water.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Almost Famous Soundtrack/10 Simple Man.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Audio News - Rock'n' Roll Forever/01 Rock Around The Clock.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Austin Powers_ International Man Of Mystery/01 The Magic Piper (Of Love).m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Austin Powers_ The Spy Who Shagged Me/04 American Woman.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Back To Dance/03 Long Cool Woman In A Black Dress.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Back To The 70's - O Album Da Década/03 American Pie.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Bambolê/09 In The Mood.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Bambolê - Volume II/03 Blue Moon.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Big Brother Brasil 2004/04 I Will Survive.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Collateral Soundtrack/03 Hands Of Time.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Forrest Gump - The Soundtrack/1-12 California Dreamin'.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Forrest Gump - The Soundtrack/1-16 Mrs. Robinson.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Ghost World - Original Motion Picture Soundtrack/01 Jaan Pechechaan Ho.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Grease [Original Soundtrack]/01 Grease.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/La Bamba/09 Summertime Blues.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Pretty Woman/10 Oh Pretty Woman.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents African Groove/01 Saye Mogo Bana.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Arabic Groove/02 Galbi.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Asian Groove/03 Remember Tomorrow.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/01 Midnight Dream.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/03 Banal Reality.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/04 Parchman Blues.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Blues Lounge/06 Run On.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Brazilian Groove/01 Maria Moita.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Brazilian Lounge/08 E Depois....m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Brazilian Lounge/11 Os Grilos.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Euro Lounge/01 Un Simple Histoire.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Euro Lounge/02 Limbe.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Euro Lounge/05 Sempre Di Domenica.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents Euro Lounge/12 Voulez-Vous_.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents World Lounge/03 Santa Maria.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents_ A New Groove/02 Dirty Laundry.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents_ Blues Around the World/02 Canceriano Sem Lar (Clinica Tobias Blues).m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents_ Euro Groove/03 Check In.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Putumayo Presents_ World Groove/01 Attention.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Saturday Night Fever/01 Stayin' Alive.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/Saturday Night Fever/03 Night Fever.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/The Best Air Guitar Album In The World... Ever!/2-06 Johnny B. Goode.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/The Full Monty - Soundtrack/02 You Sexy Thing.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Compilations/The Full Monty - Soundtrack/11 We Are Family.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Cut Copy/Zonoscope (Bonus Version)/10 Corner of the Sky.m4a",
	"/Music/iTunes 1/iTunes Media/Music/David Bowie/Changesbowie/07 Diamond Dogs.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Douster & Savage Skulls/Get Rich or High Tryin' - EP/01 Bad Gal.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Elton John/Greatest Hits 1970-2002/1-04 Rocket Man (I Think It's Going to Be a Long, Long Time).m4a",
	"/Music/iTunes 1/iTunes Media/Music/Elvis Presley/ELV1S 30 #1 Hits/02 Don't Be Cruel.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Eric Clapton/The Cream Of Clapton/03 I Feel Free.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Fleetwood Mac/The Very Best Of Fleetwood Mac/02 Don't Stop.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Françoise Hardy/Comment te dire adieu/Comment te dire adieu.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Games/That We Can Play - EP/01 Strawberry Skies.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Grand Funk Railroad/Collectors Series/The Loco-Motion.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Henry Mancini/The Pink Panther (Music from the Film Score)/The Pink Panther Theme.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Holy Ghost!/Do It Again - Single/01 Do It Again.m4a",
	"/Music/iTunes 1/iTunes Media/Music/K.C. & The Sunshine Band/The Best of/03 I'm Your Boogie Man.m4a",
	"/Music/iTunes 1/iTunes Media/Music/K.C. & The Sunshine Band/Unknown Album/Megamix (Thats The Way, Shake Your Booty, Get Down Tonight, Give It Up).m4a",
	"/Music/iTunes 1/iTunes Media/Music/Kim Ann Foxman & Andy Butler/Creature - EP/01 Creature.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Nico/Chelsea Girl/01 The Fairest Of The Seasons.m4a",
	"/Music/iTunes 1/iTunes Media/Music/oOoOO/oOoOO - EP/02 Burnout Eyess.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Peter Frampton/The Very Best of Peter Frampton/Baby, I Love Your Way.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Peter Frampton/The Very Best of Peter Frampton/Show Me The Way.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Raul Seixas/A Arte De Raul Seixas/03 Metamorfose Ambulante.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Raul Seixas/A Arte De Raul Seixas/18 Eu Nasci há 10 Mil Anos Atrás.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Rick James/Street Songs/Super Freak.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Rita Lee/Fruto Proibido/Agora Só Falta Você.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Rita Lee/Fruto Proibido/Esse Tal De Roque Enrow.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Roberto Carlos/Roberto Carlos 1966/05 Negro Gato.m4a",
	"/Music/iTunes 1/iTunes Media/Music/SOHO/Goddess/02 Hippychick.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Stan Getz/Getz_Gilberto/05 Corcovado (Quiet Nights of Quiet Stars).m4a",
	"/Music/iTunes 1/iTunes Media/Music/Steely Dan/Pretzel Logic/Rikki Don't Loose That Number.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Stevie Wonder/For Once In My Life/I Don't Know Why.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Teebs/Ardour/While You Doooo.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Beatles/Magical Mystery Tour/08 Strawberry Fields Forever.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Beatles/Past Masters, Vol. 1/10 Long Tall Sally.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Beatles/Please Please Me/14 Twist And Shout.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Beatles/Sgt. Pepper's Lonely Hearts Club Band/03 Lucy In The Sky With Diamonds.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Black Crowes/Amorica/09 Wiser Time.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Black Crowes/By Your Side/05 Only A Fool.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Black Crowes/Shake Your Money Maker/04 Could I''ve Been So Blind.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Black Crowes/The Southern Harmony And Musical Companion/01 Sting Me.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Black Crowes/Three Snakes And One Charm/02 Good Friday.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Doors/Strange Days (40th Anniversary Mixes)/01 Strange Days.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Rolling Stones/Forty Licks/1-03 (I Can't Get No) Satisfaction.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Velvet Underground/The Velvet Underground & Nico/02 I'm Waiting For The Man.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Velvet Underground/The Velvet Underground & Nico/03 Femme Fatale.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Velvet Underground/White Light_White Heat/04 Here She Comes Now.m4a",
	"/Music/iTunes 1/iTunes Media/Music/The Who/Sings My Generation/My Generation.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Village People/The Very Best Of Village People/Macho Man.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Vondelpark/Sauna - EP/01 California Analog Dream.m4a",
	"/Music/iTunes 1/iTunes Media/Music/War/Why Can't We Be Friends/Low Rider.m4a",
	"/Music/iTunes 1/iTunes Media/Music/Yes/Fragile/01 Roundabout.m4a",
}
