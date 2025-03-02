# Multilingual Speech-Dispatcher Client

This repository is a simple proof of concept for a bilingual speech dispatcher client. It reads in data from stdin and then uses [lingua](https://github.com/pemistahl/lingua-go) to disambiguate the language based on the languages which are defined in the user's cli flags. As a result, you can turn any synthesizer into a bilingual one assuming you have voices for the proper languages installed and lingua can disambiguate them. 

## Usage

- For a list of languages this program supports, run this program with the `--list-languages` flag. 
    - Run `go run . --help` for usage instructions
- [Bilingual conversation example](./example/bilingual_conversation.wav). Note that the example output uses espeak-ng and thus sounds robotic, but given this uses speech-dispatcher, you can use any other synthesizer you prefer. 
- [Example script](./example/script.sh) using this program with piped stdin

## Context

Bilingual speech on screen readers is a non-trivial problem. In order to determine in what language certain text should be spoken you need to either use heuristics based on unicode character codes, or use specialized NLP models which work on very small context lengths. For instance, a user using a screen reader is allowed to read just a word or two at a time; with such a small context window it is much harder to determine the intended language. Additionally. any disambiguation algorithm needs to run nearly instantly or stream in real time, even if the user passes in an entire document all at once, so as to not slow down the synthesizer. 

### Takeaways from this Experiment

Most users will only be concerned with a very small subset of languages. As such, if these languages can be defined in a config file you can get drastically better accuracy. For certain combinations like English vs Simplified Chinese where none of the writing system is shared, it is best to use a heuristic and skip any local NLP all together. 

The main issues arise when trying to disambiguate languages with [homographs](https://en.wikipedia.org/wiki/Homograph). These words are syntactically valid and could be pronounced in more than one language and still be correct in some situation. In this case you essentially need to add more context into the NLP model, otherwise it is too unreliable. As a result, it could be helpful to pass additional information like the previous or following sentence into the client, even if they aren't intended to be spoken, in order to give the model more context. You can also use metadata about the document like the header information in HTML, but many sites will not define this properly. Since any sentence could be composed to multiple languages, it appears not practical to use caching.

In summary, it appears that it is best to have this type of feature be opt-in in a screen reader scenario since there is essentially no way to get around the fact that it is an extra CPU/memory burden to be constantly running a small NLP model before every speech-dispatcher client request. That being said, if you are disambiguating between Russian and English or another two languages with different writing systems, it won't even load the model and thus dramatically reduces memory impact to a trivial level. 

## Future Improvements

If you are a developer looking to iterate on this, it is possible to make a series of improvements based on my initial experiment here: 

- Currently this program scans over each new line in the stdin data and finds the start/end index of each language block, but it is possible to experiment with different ways of buffering / streaming input data through the NLP model with lingua
    - lingua disambiguates based on n-gram probabilities so the more context you can give it the better. This comes at the cost of passing in more input data. If the document has another sentence aftwards, doing a short lookahead or storing the previous sentence context could potentially increase accuracy.
- It is possible to send multiple messages with different languages in the same SSIP block as defined [here](https://htmlpreview.github.io/?https://github.com/brailcom/speechd/blob/master/doc/ssip.html#Blocks-of-Messages-Commands). Doing so will reduce the round trip time for contacting the speech-dispatcher server. This improves the semantics of the client requests by signifying that both language requests are part of the same logical unit. 
- If your user only wants to disambiguate a small subset of languages like Simplified Chinese vs English, you can simply check the unicode values and skip using lingua or other NLP heuristics. 
- lingua has a fairly high memory burden since it needs to load the language models; you can experiment with different ways of loading them but unfortunately there is no way to get around the fact that it will be an extra resource hog (without going in and optimizing the underlying model)