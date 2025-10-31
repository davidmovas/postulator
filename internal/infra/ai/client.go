package ai

import "context"

type IClient interface {
	GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

var _ IClient = (*Client)(nil)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// TODO: TEMP CONTENT FOR TEST
	content := `
	<!-- wp:paragraph -->
<p>The pursuit of health is a fundamental human endeavor, and at the heart of this journey lies the powerful, symbiotic relationship between physical activity and overall well-being. Sport and exercise are not merely tools for aesthetic improvement; they are foundational pillars for a vibrant, functional, and fulfilling life. In our increasingly sedentary world, understanding and embracing the multifaceted benefits of an active lifestyle is more critical than ever.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>The impact extends far beyond the physical, weaving into the very fabric of our mental and emotional resilience.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Physical Health Benefits</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>Physically, the advantages of regular sport participation are profound and well-documented. The most obvious benefit is the improvement in cardiovascular health. Engaging in activities like running, swimming, or cycling strengthens the heart muscle, making it more efficient at pumping blood throughout the body. This enhanced efficiency lowers resting heart rate and blood pressure, significantly reducing the risk of heart disease, stroke, and other cardiovascular complications.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>Furthermore, physical activity is a key regulator of metabolic function. It helps the body manage blood sugar levels more effectively, increasing insulin sensitivity and playing a crucial role in preventing and managing type 2 diabetes. It also aids in maintaining a healthy weight by burning calories and boosting metabolism, which in turn reduces strain on joints and organs.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Strength and Mobility</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>Alongside internal health, sport is instrumental in building and maintaining a robust musculoskeletal system. Weight-bearing and resistance exercises, such as weightlifting or bodyweight training, stimulate muscle growth and increase bone density. This is vital for long-term mobility, balance, and independence, particularly as we age.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>Strong muscles and bones protect against injuries from falls and help prevent conditions like osteoporosis and sarcopenia. The functional strength gained from sports translates directly into everyday life, making tasks easier and reducing physical fatigue. The body becomes not just a vessel, but a capable and resilient instrument.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Mental Well-being</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>However, to focus solely on the physical aspects would be to overlook one of the most powerful benefits of sport: its impact on mental health. Engaging in physical activity is a potent antidote to stress, anxiety, and depression. During exercise, the brain releases a cascade of chemicals, including endorphins, often referred to as "feel-good" hormones.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>These endorphins act as natural mood elevators and painkillers, creating a phenomenon commonly known as the "runner's high." This biochemical shift can alleviate feelings of sadness and tension, promoting a state of relaxation and well-being long after the activity has ended. Regular participation in sport has been shown to be as effective as medication for some individuals dealing with mild to moderate depression.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Cognitive Benefits and Personal Growth</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>Moreover, sport cultivates mental fortitude and cognitive function. The challenges inherent in athletic pursuit—pushing through fatigue, mastering a new skill, coping with loss, and striving for a goal—build character. Participants learn discipline, patience, perseverance, and resilience.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>These qualities, forged on the track, court, or gym floor, are directly transferable to personal and professional life. Simultaneously, physical activity increases blood flow to the brain, which can enhance memory, sharpen concentration, and stimulate creativity. It is a natural cognitive enhancer, protecting against age-related cognitive decline and improving overall brain health.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>The Social Dimension</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>Beyond the individual, sport possesses a unique social dimension that is essential for human well-being. Team sports, in particular, foster a profound sense of community, belonging, and camaraderie. They teach invaluable lessons in teamwork, communication, and trust.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>Being part of a team provides a built-in support network, a group of individuals who share a common goal and can offer encouragement during setbacks and celebrate successes together. This social connection is a powerful buffer against loneliness and isolation, contributing significantly to emotional health. Even individual sports practiced in a club or class setting can provide this crucial social interaction and a sense of shared purpose.</p>
<!-- /wp:paragraph -->

<!-- wp:heading -->
<h2>Conclusion</h2>
<!-- /wp:heading -->

<!-- wp:paragraph -->
<p>In conclusion, the integration of sport and physical activity into daily life is a non-negotiable component of holistic health. It is a comprehensive strategy that fortifies the body against disease, sharpens the mind against decline, and nourishes the spirit against the strains of modern life.</p>
<!-- /wp:paragraph -->

<!-- wp:paragraph -->
<p>From the powerful heart and strong bones to the clear, resilient mind and the sense of community, the rewards are immense and interconnected. Embracing an active lifestyle is ultimately a profound commitment to oneself—a commitment to living not just longer, but with greater vitality, purpose, and joy.</p>
<!-- /wp:paragraph -->
	`

	return content, nil
}
